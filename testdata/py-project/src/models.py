from dataclasses import dataclass
from typing import Optional


@dataclass
class User:
    id: str
    name: str
    email: str
    active: bool = True


@dataclass
class Product:
    id: str
    name: str
    price: float


def create_user(name: str, email: str) -> "User":
    return User(id="", name=name, email=email)


def _internal_helper(x):
    return x
