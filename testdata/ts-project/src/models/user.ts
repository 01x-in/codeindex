import { Logger } from '../utils';

export interface User {
  id: number;
  name: string;
  email: string;
}

export class UserService {
  private logger: Logger;

  constructor(logger: Logger) {
    this.logger = logger;
  }

  getUser(id: number): User {
    this.logger.info(`Getting user ${id}`);
    return { id, name: 'Test', email: 'test@example.com' };
  }

  createUser(name: string, email: string): User {
    this.logger.info(`Creating user ${name}`);
    return { id: Date.now(), name, email };
  }
}
