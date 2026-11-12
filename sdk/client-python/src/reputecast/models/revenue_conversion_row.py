from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.revenue_conversion_row_kind import RevenueConversionRowKind






T = TypeVar("T", bound="RevenueConversionRow")



@_attrs_define
class RevenueConversionRow:
    """ 
        Attributes:
            by_email (int):
            by_token (int):
            count (int):
            id (str):
            kind (RevenueConversionRowKind):
            name (str):
            value_cents (int):
     """

    by_email: int
    by_token: int
    count: int
    id: str
    kind: RevenueConversionRowKind
    name: str
    value_cents: int





    def to_dict(self) -> dict[str, Any]:
        by_email = self.by_email

        by_token = self.by_token

        count = self.count

        id = self.id

        kind = self.kind.value

        name = self.name

        value_cents = self.value_cents


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "byEmail": by_email,
            "byToken": by_token,
            "count": count,
            "id": id,
            "kind": kind,
            "name": name,
            "valueCents": value_cents,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        by_email = d.pop("byEmail")

        by_token = d.pop("byToken")

        count = d.pop("count")

        id = d.pop("id")

        kind = RevenueConversionRowKind(d.pop("kind"))




        name = d.pop("name")

        value_cents = d.pop("valueCents")

        revenue_conversion_row = cls(
            by_email=by_email,
            by_token=by_token,
            count=count,
            id=id,
            kind=kind,
            name=name,
            value_cents=value_cents,
        )

        return revenue_conversion_row

