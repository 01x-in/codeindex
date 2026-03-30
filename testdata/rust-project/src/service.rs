use crate::models::{User, Status};

pub struct UserService {
    users: std::collections::HashMap<String, User>,
}

impl UserService {
    pub fn new() -> Self {
        UserService {
            users: std::collections::HashMap::new(),
        }
    }

    pub fn create_user(&mut self, name: String, email: String) -> User {
        let id = generate_id();
        let user = User { id: id.clone(), name, email };
        self.users.insert(id, user.clone());
        user
    }

    pub fn get_user(&self, id: &str) -> Option<&User> {
        self.users.get(id)
    }
}

fn generate_id() -> String {
    format!("{}", std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .unwrap()
        .subsec_nanos())
}
