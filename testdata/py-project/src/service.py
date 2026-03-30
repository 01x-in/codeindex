from src.models import User, Product, create_user
from src.utils import generate_id, format_name
from typing import Optional


class UserService:
    def __init__(self):
        self.users = {}

    def create(self, name: str, email: str) -> User:
        user_id = generate_id()
        user = create_user(name, email)
        user.id = user_id
        self.users[user_id] = user
        return user

    def get(self, user_id: str) -> Optional[User]:
        return self.users.get(user_id)
