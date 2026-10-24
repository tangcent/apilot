package com.example.api;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import com.example.model.Wallet;

/**
 * Handler whose package also declares a Wallet. The single-type import names
 * the model one, which per JLS 6.4.1 shadows the same-package declaration.
 */
@RestController
@RequestMapping("/api/wallet")
public class WalletController {

    @GetMapping("/me")
    public Wallet getWallet() {
        return null;
    }
}
