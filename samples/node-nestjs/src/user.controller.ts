import { Controller, Get, Post, Put, Delete, Patch, Param, Body, Query } from '@nestjs/common';

@Controller('users')
export class UserController {
    @Get()
    listUsers(@Query('name') name: string, @Query('role') role: string = 'user') {
        return { users: [] };
    }

    @Post()
    createUser(@Body() req: CreateUserReq) {
        return req;
    }

    @Get(':id')
    getUser(@Param('id') id: string) {
        return { id };
    }

    @Put(':id')
    updateUser(@Param('id') id: string, @Body() req: UpdateUserReq) {
        return req;
    }

    @Delete(':id')
    deleteUser(@Param('id') id: string) {
        return '';
    }

    @Patch(':id')
    patchUser(@Param('id') id: string, @Query('name') name: string = 'unknown') {
        return { id };
    }
}

class CreateUserReq {
    name: string;
    email: string;
}

class UpdateUserReq {
    name: string;
    email: string;
}
