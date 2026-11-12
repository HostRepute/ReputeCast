from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="EarnMemberDiscount")



@_attrs_define
class EarnMemberDiscount:
    """ 
        Attributes:
            amount_cents (int):
            label (str):
            duration (str | Unset):
            ends_at (datetime.datetime | Unset):
            percent (float | Unset):
     """

    amount_cents: int
    label: str
    duration: str | Unset = UNSET
    ends_at: datetime.datetime | Unset = UNSET
    percent: float | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        amount_cents = self.amount_cents

        label = self.label

        duration = self.duration

        ends_at: str | Unset = UNSET
        if not isinstance(self.ends_at, Unset):
            ends_at = self.ends_at.isoformat()

        percent = self.percent


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "amountCents": amount_cents,
            "label": label,
        })
        if duration is not UNSET:
            field_dict["duration"] = duration
        if ends_at is not UNSET:
            field_dict["endsAt"] = ends_at
        if percent is not UNSET:
            field_dict["percent"] = percent

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        amount_cents = d.pop("amountCents")

        label = d.pop("label")

        duration = d.pop("duration", UNSET)

        _ends_at = d.pop("endsAt", UNSET)
        ends_at: datetime.datetime | Unset
        if isinstance(_ends_at,  Unset):
            ends_at = UNSET
        else:
            ends_at = datetime.datetime.fromisoformat(_ends_at)




        percent = d.pop("percent", UNSET)

        earn_member_discount = cls(
            amount_cents=amount_cents,
            label=label,
            duration=duration,
            ends_at=ends_at,
            percent=percent,
        )

        return earn_member_discount

