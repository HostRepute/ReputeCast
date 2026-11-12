from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.products_payments_mode import ProductsPaymentsMode






T = TypeVar("T", bound="ProductsPayments")



@_attrs_define
class ProductsPayments:
    """ 
        Attributes:
            account (str):
            connected (bool):
            currency (str):
            has_live (bool):
            mode (ProductsPaymentsMode):
            prices_include_tax (bool):
            provider (str):
            tax_at_checkout (bool):
     """

    account: str
    connected: bool
    currency: str
    has_live: bool
    mode: ProductsPaymentsMode
    prices_include_tax: bool
    provider: str
    tax_at_checkout: bool





    def to_dict(self) -> dict[str, Any]:
        account = self.account

        connected = self.connected

        currency = self.currency

        has_live = self.has_live

        mode = self.mode.value

        prices_include_tax = self.prices_include_tax

        provider = self.provider

        tax_at_checkout = self.tax_at_checkout


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "account": account,
            "connected": connected,
            "currency": currency,
            "hasLive": has_live,
            "mode": mode,
            "pricesIncludeTax": prices_include_tax,
            "provider": provider,
            "taxAtCheckout": tax_at_checkout,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        account = d.pop("account")

        connected = d.pop("connected")

        currency = d.pop("currency")

        has_live = d.pop("hasLive")

        mode = ProductsPaymentsMode(d.pop("mode"))




        prices_include_tax = d.pop("pricesIncludeTax")

        provider = d.pop("provider")

        tax_at_checkout = d.pop("taxAtCheckout")

        products_payments = cls(
            account=account,
            connected=connected,
            currency=currency,
            has_live=has_live,
            mode=mode,
            prices_include_tax=prices_include_tax,
            provider=provider,
            tax_at_checkout=tax_at_checkout,
        )

        return products_payments

