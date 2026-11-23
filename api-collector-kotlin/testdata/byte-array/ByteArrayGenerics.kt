package com.example.bytearray

import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

/** Takes its type argument as a field, an element, an array and a map value. */
class ByteArrayWrapper<U>(
    val data: U?,
    val items: List<U>?,
    val grid: Array<U>?,
    val byName: Map<String, U>?,
)

/** Hands its type argument on to a generic class, as a field and as an element. */
class ByteArrayOuter<T>(
    val wrapped: ByteArrayWrapper<T>?,
    val wrappedItems: List<ByteArrayWrapper<T>>?,
    val direct: T?,
)

open class ByteArrayBase<B>(
    val payload: B?,
    val wrapped: ByteArrayWrapper<B>?,
)

/** The superclass argument is a byte array. */
class ByteArraySub : ByteArrayBase<ByteArray>(null, null)

/** Boxed bytes are numbers, whichever class passes them on. */
class ByteArrayBoxedSub : ByteArrayBase<Array<Byte>>(null, null)

/** The superclass argument is a char array. */
class ByteArrayCharSub : ByteArrayBase<CharArray>(null, null)

/** Passes its own type argument to the superclass. */
class ByteArrayOpenSub<X> : ByteArrayBase<X>(null, null)

class ByteArrayResult<T>(
    val code: Int,
    val data: T?,
)

abstract class ByteArrayBaseController<T> {

    @PostMapping("/echo")
    fun echo(@RequestBody body: T): ByteArrayResult<T> = ByteArrayResult(0, body)
}

@RestController
@RequestMapping("/byte-arrays/typed")
class ByteArrayTypedController : ByteArrayBaseController<ByteArray>()

@RestController
@RequestMapping("/byte-arrays/typed-chars")
class ByteArrayCharController : ByteArrayBaseController<CharArray>()
