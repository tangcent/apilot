import { Controller, Get, Post, Param, Body } from '@nestjs/common';

/**
 * OrderController manages order operations.
 */
@Controller('orders')
export class OrderController {

  /**
   * listOrders returns all orders.
   */
  @Get()
  listOrders(): string {
    return 'list';
  }

  /**
   * createOrder creates a new order.
   */
  @Post()
  createOrder(@Body() body: any): string {
    return 'created';
  }

  /**
   * getOrder returns a single order by ID.
   */
  @Get(':id')
  getOrder(@Param('id') id: string): string {
    return 'order';
  }
}
