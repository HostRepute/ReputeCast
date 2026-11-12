from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="RevenueSourceRow")



@_attrs_define
class RevenueSourceRow:
    """ 
        Attributes:
            at_start (int):
            base (int):
            joined (int):
            key (str):
            kind (str):
            name (str):
            paid (int):
            too_few (bool):
            form_kind (str | Unset):
            others (int | Unset):
            rate (float | Unset):
     """

    at_start: int
    base: int
    joined: int
    key: str
    kind: str
    name: str
    paid: int
    too_few: bool
    form_kind: str | Unset = UNSET
    others: int | Unset = UNSET
    rate: float | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        at_start = self.at_start

        base = self.base

        joined = self.joined

        key = self.key

        kind = self.kind

        name = self.name

        paid = self.paid

        too_few = self.too_few

        form_kind = self.form_kind

        others = self.others

        rate = self.rate


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "atStart": at_start,
            "base": base,
            "joined": joined,
            "key": key,
            "kind": kind,
            "name": name,
            "paid": paid,
            "tooFew": too_few,
        })
        if form_kind is not UNSET:
            field_dict["formKind"] = form_kind
        if others is not UNSET:
            field_dict["others"] = others
        if rate is not UNSET:
            field_dict["rate"] = rate

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at_start = d.pop("atStart")

        base = d.pop("base")

        joined = d.pop("joined")

        key = d.pop("key")

        kind = d.pop("kind")

        name = d.pop("name")

        paid = d.pop("paid")

        too_few = d.pop("tooFew")

        form_kind = d.pop("formKind", UNSET)

        others = d.pop("others", UNSET)

        rate = d.pop("rate", UNSET)

        revenue_source_row = cls(
            at_start=at_start,
            base=base,
            joined=joined,
            key=key,
            kind=kind,
            name=name,
            paid=paid,
            too_few=too_few,
            form_kind=form_kind,
            others=others,
            rate=rate,
        )

        return revenue_source_row

