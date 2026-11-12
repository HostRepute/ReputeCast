from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="DeliverabilityPostmasterPoint")



@_attrs_define
class DeliverabilityPostmasterPoint:
    """ 
        Attributes:
            from_ (datetime.date):
            to (datetime.date):
            spam_rate (float | Unset):
     """

    from_: datetime.date
    to: datetime.date
    spam_rate: float | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from_ = self.from_.isoformat()

        to = self.to.isoformat()

        spam_rate = self.spam_rate


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "from": from_,
            "to": to,
        })
        if spam_rate is not UNSET:
            field_dict["spamRate"] = spam_rate

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        from_ = datetime.date.fromisoformat(d.pop("from"))




        to = datetime.date.fromisoformat(d.pop("to"))




        spam_rate = d.pop("spamRate", UNSET)

        deliverability_postmaster_point = cls(
            from_=from_,
            to=to,
            spam_rate=spam_rate,
        )

        return deliverability_postmaster_point

