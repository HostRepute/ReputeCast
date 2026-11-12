from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.contact_paid_status import ContactPaidStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ContactPaid")



@_attrs_define
class ContactPaid:
    """ 
        Attributes:
            member_id (str):
            status (ContactPaidStatus):
            tier (str): The tier's short name (Insider)
            ended_at (datetime.datetime | Unset):
     """

    member_id: str
    status: ContactPaidStatus
    tier: str
    ended_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        member_id = self.member_id

        status = self.status.value

        tier = self.tier

        ended_at: str | Unset = UNSET
        if not isinstance(self.ended_at, Unset):
            ended_at = self.ended_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "memberId": member_id,
            "status": status,
            "tier": tier,
        })
        if ended_at is not UNSET:
            field_dict["endedAt"] = ended_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        member_id = d.pop("memberId")

        status = ContactPaidStatus(d.pop("status"))




        tier = d.pop("tier")

        _ended_at = d.pop("endedAt", UNSET)
        ended_at: datetime.datetime | Unset
        if isinstance(_ended_at,  Unset):
            ended_at = UNSET
        else:
            ended_at = datetime.datetime.fromisoformat(_ended_at)




        contact_paid = cls(
            member_id=member_id,
            status=status,
            tier=tier,
            ended_at=ended_at,
        )

        return contact_paid

