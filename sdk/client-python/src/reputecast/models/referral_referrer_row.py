from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.referral_referrer_row_link_status import ReferralReferrerRowLinkStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ReferralReferrerRow")



@_attrs_define
class ReferralReferrerRow:
    """ 
        Attributes:
            contact_id (str):
            counted (int):
            email (str):
            held (int):
            in_hold (int):
            link_id (str):
            link_status (ReferralReferrerRowLinkStatus):
            milestones (list[int]):
            name (str):
            not_counted (int):
            rewards_sent (int):
            unclaimed (str):
            last_referral (datetime.datetime | Unset):
     """

    contact_id: str
    counted: int
    email: str
    held: int
    in_hold: int
    link_id: str
    link_status: ReferralReferrerRowLinkStatus
    milestones: list[int]
    name: str
    not_counted: int
    rewards_sent: int
    unclaimed: str
    last_referral: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        contact_id = self.contact_id

        counted = self.counted

        email = self.email

        held = self.held

        in_hold = self.in_hold

        link_id = self.link_id

        link_status = self.link_status.value

        milestones = self.milestones



        name = self.name

        not_counted = self.not_counted

        rewards_sent = self.rewards_sent

        unclaimed = self.unclaimed

        last_referral: str | Unset = UNSET
        if not isinstance(self.last_referral, Unset):
            last_referral = self.last_referral.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "contactId": contact_id,
            "counted": counted,
            "email": email,
            "held": held,
            "inHold": in_hold,
            "linkId": link_id,
            "linkStatus": link_status,
            "milestones": milestones,
            "name": name,
            "notCounted": not_counted,
            "rewardsSent": rewards_sent,
            "unclaimed": unclaimed,
        })
        if last_referral is not UNSET:
            field_dict["lastReferral"] = last_referral

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        contact_id = d.pop("contactId")

        counted = d.pop("counted")

        email = d.pop("email")

        held = d.pop("held")

        in_hold = d.pop("inHold")

        link_id = d.pop("linkId")

        link_status = ReferralReferrerRowLinkStatus(d.pop("linkStatus"))




        milestones = cast(list[int], d.pop("milestones"))


        name = d.pop("name")

        not_counted = d.pop("notCounted")

        rewards_sent = d.pop("rewardsSent")

        unclaimed = d.pop("unclaimed")

        _last_referral = d.pop("lastReferral", UNSET)
        last_referral: datetime.datetime | Unset
        if isinstance(_last_referral,  Unset):
            last_referral = UNSET
        else:
            last_referral = datetime.datetime.fromisoformat(_last_referral)




        referral_referrer_row = cls(
            contact_id=contact_id,
            counted=counted,
            email=email,
            held=held,
            in_hold=in_hold,
            link_id=link_id,
            link_status=link_status,
            milestones=milestones,
            name=name,
            not_counted=not_counted,
            rewards_sent=rewards_sent,
            unclaimed=unclaimed,
            last_referral=last_referral,
        )

        return referral_referrer_row

