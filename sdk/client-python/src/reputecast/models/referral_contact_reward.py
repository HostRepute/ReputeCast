from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ReferralContactReward")



@_attrs_define
class ReferralContactReward:
    """ 
        Attributes:
            milestone (int):
            name (str):
            status (str):
            sent_at (datetime.datetime | Unset):
            used_at (datetime.datetime | Unset):
     """

    milestone: int
    name: str
    status: str
    sent_at: datetime.datetime | Unset = UNSET
    used_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        milestone = self.milestone

        name = self.name

        status = self.status

        sent_at: str | Unset = UNSET
        if not isinstance(self.sent_at, Unset):
            sent_at = self.sent_at.isoformat()

        used_at: str | Unset = UNSET
        if not isinstance(self.used_at, Unset):
            used_at = self.used_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "milestone": milestone,
            "name": name,
            "status": status,
        })
        if sent_at is not UNSET:
            field_dict["sentAt"] = sent_at
        if used_at is not UNSET:
            field_dict["usedAt"] = used_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        milestone = d.pop("milestone")

        name = d.pop("name")

        status = d.pop("status")

        _sent_at = d.pop("sentAt", UNSET)
        sent_at: datetime.datetime | Unset
        if isinstance(_sent_at,  Unset):
            sent_at = UNSET
        else:
            sent_at = datetime.datetime.fromisoformat(_sent_at)




        _used_at = d.pop("usedAt", UNSET)
        used_at: datetime.datetime | Unset
        if isinstance(_used_at,  Unset):
            used_at = UNSET
        else:
            used_at = datetime.datetime.fromisoformat(_used_at)




        referral_contact_reward = cls(
            milestone=milestone,
            name=name,
            status=status,
            sent_at=sent_at,
            used_at=used_at,
        )

        return referral_contact_reward

