use axum::{
    extract::Multipart,
    response::IntoResponse,
    routing::post,
    Router,
};
use git2::{Cred, PushOptions, RemoteCallbacks, Repository};
use std::net::SocketAddr;
use tempfile::TempDir;
use thiserror::Error;
use tower_http::{
    services::ServeDir,
    trace::{DefaultMakeSpan, TraceLayer},
};
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt};

#[derive(Error, Debug)]
pub enum UploadError {
    #[error("Git error: {0}")]
    Git(#[from] git2::Error),
    #[error("IO error: {0}")]
    Io(#[from] std::io::Error),
    #[error("Missing environment variable: {0}")]
    EnvVar(String),
}

async fn handle_upload(mut multipart: Multipart) -> Result<impl IntoResponse, UploadError> {
    // Create a temporary directory for git operations
    let temp_dir = TempDir::new()?;
    let repo_path = temp_dir.path();

    // Initialize git repo
    let repo = if let Some(gist_id) = std::env::var("GIST_ID").ok() {
        // Clone existing gist
        clone_gist(repo_path, &gist_id)?
    } else {
        // Create new gist
        init_new_gist(repo_path)?
    };

    while let Some(field) = multipart.next_field().await.unwrap() {
        let file_name = field.file_name().unwrap().to_string();
        let data = field.bytes().await.unwrap();
        
        // Write file to git repo
        let file_path = repo_path.join(&file_name);
        std::fs::write(&file_path, data)?;

        // Stage the file
        let mut index = repo.index()?;
        index.add_path(std::path::Path::new(&file_name))?;
        index.write()?;

        // Create commit
        let tree_id = index.write_tree()?;
        let tree = repo.find_tree(tree_id)?;
        
        let sig = repo.signature()?;
        if let Ok(parent) = repo.head() {
            let parent_commit = repo.find_commit(parent.target().unwrap())?;
            repo.commit(
                Some("HEAD"),
                &sig,
                &sig,
                &format!("Add {}", file_name),
                &tree,
                &[&parent_commit],
            )?;
        } else {
            repo.commit(
                Some("HEAD"),
                &sig,
                &sig,
                &format!("Initial commit: {}", file_name),
                &tree,
                &[],
            )?;
        }
    }

    // Push changes
    push_changes(&repo)?;

    Ok("File uploaded successfully")
}

fn clone_gist(path: &std::path::Path, gist_id: &str) -> Result<Repository, git2::Error> {
    let mut callbacks = RemoteCallbacks::new();
    callbacks.credentials(|_url, username_from_url, _allowed_types| {
        Cred::ssh_key(
            username_from_url.unwrap(),
            None,
            std::path::Path::new(&std::env::var("SSH_KEY_PATH").unwrap_or_else(|_| String::from("/root/.ssh/id_ed25519"))),
            None,
        )
    });

    let mut fetch_options = git2::FetchOptions::new();
    fetch_options.remote_callbacks(callbacks);

    let mut builder = git2::build::RepoBuilder::new();
    builder.fetch_options(fetch_options);

    builder.clone(
        &format!("git@gist.github.com:{}.git", gist_id),
        path,
    )
}

fn init_new_gist(path: &std::path::Path) -> Result<Repository, git2::Error> {
    let repo = Repository::init(path)?;
    repo.remote(
        "origin",
        "git@gist.github.com:new.git",
    )?;
    Ok(repo)
}

fn push_changes(repo: &Repository) -> Result<(), git2::Error> {
    let mut callbacks = RemoteCallbacks::new();
    callbacks.credentials(|_url, username_from_url, _allowed_types| {
        Cred::ssh_key(
            username_from_url.unwrap(),
            None,
            std::path::Path::new(&std::env::var("SSH_KEY_PATH").unwrap_or_else(|_| String::from("/root/.ssh/id_ed25519"))),
            None,
        )
    });

    let mut push_options = PushOptions::new();
    push_options.remote_callbacks(callbacks);

    let mut remote = repo.find_remote("origin")?;
    remote.push(&["refs/heads/master:refs/heads/master"], Some(&mut push_options))
}

#[tokio::main]
async fn main() {
    // Initialize tracing
    tracing_subscriber::registry()
        .with(tracing_subscriber::EnvFilter::new(
            std::env::var("RUST_LOG").unwrap_or_else(|_| "info".into()),
        ))
        .with(tracing_subscriber::fmt::layer())
        .init();

    // Build our application with routes
    let app = Router::new()
        .route("/upload", post(handle_upload))
        .nest_service("/", ServeDir::new("static"))
        .layer(
            TraceLayer::new_for_http()
                .make_span_with(DefaultMakeSpan::default()
                    .include_headers(true)),
        );

    // Run it
    let addr = SocketAddr::from(([0, 0, 0, 0], 3000));
    tracing::info!("listening on {}", addr);
    hyper::Server::bind(&addr)
        .serve(app.into_make_service())
        .await
        .unwrap();
}
