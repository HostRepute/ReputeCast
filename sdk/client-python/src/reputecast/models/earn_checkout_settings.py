from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.earn_portal import EarnPortal





T = TypeVar("T", bound="EarnCheckoutSettings")



@_attrs_define
class EarnCheckoutSettings:
    """ 
        Attributes:
            currency (str):
            currency_fixed (bool):
            portal (EarnPortal):
            prices_include_tax (bool):
            tax_at_checkout (bool):
            tax_available (bool):
     """

    currency: str
    currency_fixed: bool
    portal: EarnPortal
    prices_include_tax: bool
    tax_at_checkout: bool
    tax_available: bool





    def to_dict(self) -> dict[str, Any]:
        from ..models.earn_portal import EarnPortal # noqa: PLC0415
        currency = self.currency

        currency_fixed = self.currency_fixed

        portal = self.portal.to_dict()

        prices_include_tax = self.prices_include_tax

        tax_at_checkout = self.tax_at_checkout

        tax_available = self.tax_available


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "currency": currency,
            "currencyFixed": currency_fixed,
            "portal": portal,
            "pricesIncludeTax": prices_include_tax,
            "taxAtCheckout": tax_at_checkout,
            "taxAvailable": tax_available,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.earn_portal import EarnPortal # noqa: PLC0415
        d = dict(src_dict)
        currency = d.pop("currency")

        currency_fixed = d.pop("currencyFixed")

        portal = EarnPortal.from_dict(d.pop("portal"))




        prices_include_tax = d.pop("pricesIncludeTax")

        tax_at_checkout = d.pop("taxAtCheckout")

        tax_available = d.pop("taxAvailable")

        earn_checkout_settings = cls(
            currency=currency,
            currency_fixed=currency_fixed,
            portal=portal,
            prices_include_tax=prices_include_tax,
            tax_at_checkout=tax_at_checkout,
            tax_available=tax_available,
        )

        return earn_checkout_settings

