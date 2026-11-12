from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.product_price_kind import ProductPriceKind






T = TypeVar("T", bound="ProductPrice")



@_attrs_define
class ProductPrice:
    """ 
        Attributes:
            amount_cents (int):
            kind (ProductPriceKind):
            minimum_cents (int):
            suggested_cents (int):
     """

    amount_cents: int
    kind: ProductPriceKind
    minimum_cents: int
    suggested_cents: int





    def to_dict(self) -> dict[str, Any]:
        amount_cents = self.amount_cents

        kind = self.kind.value

        minimum_cents = self.minimum_cents

        suggested_cents = self.suggested_cents


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "amountCents": amount_cents,
            "kind": kind,
            "minimumCents": minimum_cents,
            "suggestedCents": suggested_cents,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        amount_cents = d.pop("amountCents")

        kind = ProductPriceKind(d.pop("kind"))




        minimum_cents = d.pop("minimumCents")

        suggested_cents = d.pop("suggestedCents")

        product_price = cls(
            amount_cents=amount_cents,
            kind=kind,
            minimum_cents=minimum_cents,
            suggested_cents=suggested_cents,
        )

        return product_price

