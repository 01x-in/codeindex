mod models;
mod service;

use service::UserService;

fn main() {
    let mut svc = UserService::new();
    let user = svc.create_user("Alice".to_string(), "alice@example.com".to_string());
    println!("Created user: {:?}", user);
}
