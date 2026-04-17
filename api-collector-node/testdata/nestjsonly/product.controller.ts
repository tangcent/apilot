import { Controller, Get, Post, Param, Query, Body } from '@nestjs/common';

/**
 * ProductController manages product operations.
 */
@Controller('products')
export class ProductController {

  /**
   * listProducts returns all products.
   */
  @Get()
  listProducts(@Query('page') page: number): string {
    return 'list';
  }

  /**
   * createProduct creates a new product.
   */
  @Post()
  createProduct(@Body() body: any): string {
    return 'created';
  }

  /**
   * getProduct returns a single product by ID.
   */
  @Get(':id')
  getProduct(@Param('id') id: string): string {
    return 'product';
  }
}
