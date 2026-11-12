from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ConversionSummary")



@_attrs_define
class ConversionSummary:
    """ 
        Attributes:
            count (int):
            credited (int):
            credited_cents (int):
            value_cents (int):
            name (str | Unset):
            sender (str | Unset):
            since (datetime.datetime | Unset):
            typical_cents (int | Unset):
     """

    count: int
    credited: int
    credited_cents: int
    value_cents: int
    name: str | Unset = UNSET
    sender: str | Unset = UNSET
    since: datetime.datetime | Unset = UNSET
    typical_cents: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        count = self.count

        credited = self.credited

        credited_cents = self.credited_cents

        value_cents = self.value_cents

        name = self.name

        sender = self.sender

        since: str | Unset = UNSET
        if not isinstance(self.since, Unset):
            since = self.since.isoformat()

        typical_cents = self.typical_cents


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "credited": credited,
            "creditedCents": credited_cents,
            "valueCents": value_cents,
        })
        if name is not UNSET:
            field_dict["name"] = name
        if sender is not UNSET:
            field_dict["sender"] = sender
        if since is not UNSET:
            field_dict["since"] = since
        if typical_cents is not UNSET:
            field_dict["typicalCents"] = typical_cents

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        count = d.pop("count")

        credited = d.pop("credited")

        credited_cents = d.pop("creditedCents")

        value_cents = d.pop("valueCents")

        name = d.pop("name", UNSET)

        sender = d.pop("sender", UNSET)

        _since = d.pop("since", UNSET)
        since: datetime.datetime | Unset
        if isinstance(_since,  Unset):
            since = UNSET
        else:
            since = datetime.datetime.fromisoformat(_since)




        typical_cents = d.pop("typicalCents", UNSET)

        conversion_summary = cls(
            count=count,
            credited=credited,
            credited_cents=credited_cents,
            value_cents=value_cents,
            name=name,
            sender=sender,
            since=since,
            typical_cents=typical_cents,
        )

        return conversion_summary

