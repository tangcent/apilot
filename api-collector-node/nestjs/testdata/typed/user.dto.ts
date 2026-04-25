import { IsString, IsNumber, IsEmail, IsOptional, IsBoolean, IsArray, IsEnum } from 'class-validator';
import { ApiProperty, ApiResponse } from '@nestjs/swagger';

export enum UserRole {
  Admin = 'admin',
  User = 'user',
  Guest = 'guest',
}

export class CreateUserDto {
  @IsString()
  @ApiProperty({ description: 'User name' })
  name: string;

  @IsEmail()
  @ApiProperty({ description: 'User email address' })
  email: string;

  @IsOptional()
  @IsNumber()
  @ApiProperty({ description: 'User age', required: false })
  age?: number;

  @IsEnum(UserRole)
  @ApiProperty({ description: 'User role', enum: UserRole })
  role: UserRole;
}

export class UpdateUserDto {
  @IsOptional()
  @IsString()
  @ApiProperty({ description: 'User name', required: false })
  name?: string;

  @IsOptional()
  @IsEmail()
  @ApiProperty({ description: 'User email address', required: false })
  email?: string;

  @IsOptional()
  @IsNumber()
  @ApiProperty({ description: 'User age', required: false })
  age?: number;
}

export class UserResponse {
  @ApiProperty({ description: 'User ID' })
  id: number;

  @ApiProperty({ description: 'User name' })
  name: string;

  @ApiProperty({ description: 'User email address' })
  email: string;

  @IsOptional()
  @ApiProperty({ description: 'User age', required: false })
  age?: number;

  @ApiProperty({ description: 'User role' })
  role: string;
}

export class ListUsersResponse {
  @ApiProperty({ description: 'List of users', type: 'array' })
  users: UserResponse[];

  @ApiProperty({ description: 'Total count' })
  total: number;
}
