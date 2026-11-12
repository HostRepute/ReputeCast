from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.page_result_kind import PageResultKind






T = TypeVar("T", bound="PageResult")



@_attrs_define
class PageResult:
    """ 
        Attributes:
            amount_cents (int):
            conversion (float):
            count (int):
            currency (str):
            kind (PageResultKind):
     """

    amount_cents: int
    conversion: float
    count: int
    currency: str
    kind: PageResultKind





    def to_dict(self) -> dict[str, Any]:
        amount_cents = self.amount_cents

        conversion = self.conversion

        count = self.count

        currency = self.currency

        kind = self.kind.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "amountCents": amount_cents,
            "conversion": conversion,
            "count": count,
            "currency": currency,
            "kind": kind,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        amount_cents = d.pop("amountCents")

        conversion = d.pop("conversion")

        count = d.pop("count")

        currency = d.pop("currency")

        kind = PageResultKind(d.pop("kind"))




        page_result = cls(
            amount_cents=amount_cents,
            conversion=conversion,
            count=count,
            currency=currency,
            kind=kind,
        )

        return page_result

