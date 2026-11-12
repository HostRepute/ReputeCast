from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.revenue_credit_kind import RevenueCreditKind






T = TypeVar("T", bound="RevenueCredit")



@_attrs_define
class RevenueCredit:
    """ 
        Attributes:
            count (int):
            id (str):
            kind (RevenueCreditKind):
            name (str):
     """

    count: int
    id: str
    kind: RevenueCreditKind
    name: str





    def to_dict(self) -> dict[str, Any]:
        count = self.count

        id = self.id

        kind = self.kind.value

        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "id": id,
            "kind": kind,
            "name": name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        count = d.pop("count")

        id = d.pop("id")

        kind = RevenueCreditKind(d.pop("kind"))




        name = d.pop("name")

        revenue_credit = cls(
            count=count,
            id=id,
            kind=kind,
            name=name,
        )

        return revenue_credit

