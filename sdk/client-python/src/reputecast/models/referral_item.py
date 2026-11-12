from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.referral_item_status import ReferralItemStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ReferralItem")



@_attrs_define
class ReferralItem:
    """ 
        Attributes:
            email (str):
            id (str):
            left_kind (str):
            number (int):
            reason (str):
            signed_up_at (datetime.datetime):
            status (ReferralItemStatus):
            confirmed_at (datetime.datetime | Unset):
            counted_at (datetime.datetime | Unset):
            counts_on (datetime.datetime | Unset):
            reminded_at (datetime.datetime | Unset):
     """

    email: str
    id: str
    left_kind: str
    number: int
    reason: str
    signed_up_at: datetime.datetime
    status: ReferralItemStatus
    confirmed_at: datetime.datetime | Unset = UNSET
    counted_at: datetime.datetime | Unset = UNSET
    counts_on: datetime.datetime | Unset = UNSET
    reminded_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        email = self.email

        id = self.id

        left_kind = self.left_kind

        number = self.number

        reason = self.reason

        signed_up_at = self.signed_up_at.isoformat()

        status = self.status.value

        confirmed_at: str | Unset = UNSET
        if not isinstance(self.confirmed_at, Unset):
            confirmed_at = self.confirmed_at.isoformat()

        counted_at: str | Unset = UNSET
        if not isinstance(self.counted_at, Unset):
            counted_at = self.counted_at.isoformat()

        counts_on: str | Unset = UNSET
        if not isinstance(self.counts_on, Unset):
            counts_on = self.counts_on.isoformat()

        reminded_at: str | Unset = UNSET
        if not isinstance(self.reminded_at, Unset):
            reminded_at = self.reminded_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "email": email,
            "id": id,
            "leftKind": left_kind,
            "number": number,
            "reason": reason,
            "signedUpAt": signed_up_at,
            "status": status,
        })
        if confirmed_at is not UNSET:
            field_dict["confirmedAt"] = confirmed_at
        if counted_at is not UNSET:
            field_dict["countedAt"] = counted_at
        if counts_on is not UNSET:
            field_dict["countsOn"] = counts_on
        if reminded_at is not UNSET:
            field_dict["remindedAt"] = reminded_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        email = d.pop("email")

        id = d.pop("id")

        left_kind = d.pop("leftKind")

        number = d.pop("number")

        reason = d.pop("reason")

        signed_up_at = datetime.datetime.fromisoformat(d.pop("signedUpAt"))




        status = ReferralItemStatus(d.pop("status"))




        _confirmed_at = d.pop("confirmedAt", UNSET)
        confirmed_at: datetime.datetime | Unset
        if isinstance(_confirmed_at,  Unset):
            confirmed_at = UNSET
        else:
            confirmed_at = datetime.datetime.fromisoformat(_confirmed_at)




        _counted_at = d.pop("countedAt", UNSET)
        counted_at: datetime.datetime | Unset
        if isinstance(_counted_at,  Unset):
            counted_at = UNSET
        else:
            counted_at = datetime.datetime.fromisoformat(_counted_at)




        _counts_on = d.pop("countsOn", UNSET)
        counts_on: datetime.datetime | Unset
        if isinstance(_counts_on,  Unset):
            counts_on = UNSET
        else:
            counts_on = datetime.datetime.fromisoformat(_counts_on)




        _reminded_at = d.pop("remindedAt", UNSET)
        reminded_at: datetime.datetime | Unset
        if isinstance(_reminded_at,  Unset):
            reminded_at = UNSET
        else:
            reminded_at = datetime.datetime.fromisoformat(_reminded_at)




        referral_item = cls(
            email=email,
            id=id,
            left_kind=left_kind,
            number=number,
            reason=reason,
            signed_up_at=signed_up_at,
            status=status,
            confirmed_at=confirmed_at,
            counted_at=counted_at,
            counts_on=counts_on,
            reminded_at=reminded_at,
        )

        return referral_item

