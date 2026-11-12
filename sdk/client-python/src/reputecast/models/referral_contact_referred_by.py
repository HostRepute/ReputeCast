from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.referral_contact_referred_by_status import ReferralContactReferredByStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ReferralContactReferredBy")



@_attrs_define
class ReferralContactReferredBy:
    """ 
        Attributes:
            link_id (str):
            program_id (str):
            reason (str):
            referrer (str):
            referrer_id (str):
            status (ReferralContactReferredByStatus):
            counted_at (datetime.datetime | Unset):
            counts_on (datetime.datetime | Unset):
     """

    link_id: str
    program_id: str
    reason: str
    referrer: str
    referrer_id: str
    status: ReferralContactReferredByStatus
    counted_at: datetime.datetime | Unset = UNSET
    counts_on: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        link_id = self.link_id

        program_id = self.program_id

        reason = self.reason

        referrer = self.referrer

        referrer_id = self.referrer_id

        status = self.status.value

        counted_at: str | Unset = UNSET
        if not isinstance(self.counted_at, Unset):
            counted_at = self.counted_at.isoformat()

        counts_on: str | Unset = UNSET
        if not isinstance(self.counts_on, Unset):
            counts_on = self.counts_on.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "linkId": link_id,
            "programId": program_id,
            "reason": reason,
            "referrer": referrer,
            "referrerId": referrer_id,
            "status": status,
        })
        if counted_at is not UNSET:
            field_dict["countedAt"] = counted_at
        if counts_on is not UNSET:
            field_dict["countsOn"] = counts_on

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        link_id = d.pop("linkId")

        program_id = d.pop("programId")

        reason = d.pop("reason")

        referrer = d.pop("referrer")

        referrer_id = d.pop("referrerId")

        status = ReferralContactReferredByStatus(d.pop("status"))




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




        referral_contact_referred_by = cls(
            link_id=link_id,
            program_id=program_id,
            reason=reason,
            referrer=referrer,
            referrer_id=referrer_id,
            status=status,
            counted_at=counted_at,
            counts_on=counts_on,
        )

        return referral_contact_referred_by

