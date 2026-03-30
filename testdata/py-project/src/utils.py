import uuid
import re

MAX_NAME_LENGTH = 100


def generate_id() -> str:
    return str(uuid.uuid4())


def format_name(name: str) -> str:
    name = name.strip()
    if len(name) > MAX_NAME_LENGTH:
        name = name[:MAX_NAME_LENGTH]
    return re.sub(r'\s+', ' ', name)
