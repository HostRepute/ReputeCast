from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ReferralTopReferrer")



@_attrs_define
class ReferralTopReferrer:
    """ 
        Attributes:
            counted (int):
            email (str):
            held (int):
            in_hold (int):
            link_id (str):
            milestones (list[int]):
            name (str):
            unclaimed (str):
            last_referral (datetime.datetime | Unset):
     """

    counted: int
    email: str
    held: int
    in_hold: int
    link_id: str
    milestones: list[int]
    name: str
    unclaimed: str
    last_referral: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        counted = self.counted

        email = self.email

        held = self.held

        in_hold = self.in_hold

        link_id = self.link_id

        milestones = self.milestones



        name = self.name

        unclaimed = self.unclaimed

        last_referral: str | Unset = UNSET
        if not isinstance(self.last_referral, Unset):
            last_referral = self.last_referral.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "counted": counted,
            "email": email,
            "held": held,
            "inHold": in_hold,
            "linkId": link_id,
            "milestones": milestones,
            "name": name,
            "unclaimed": unclaimed,
        })
        if last_referral is not UNSET:
            field_dict["lastReferral"] = last_referral

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        counted = d.pop("counted")

        email = d.pop("email")

        held = d.pop("held")

        in_hold = d.pop("inHold")

        link_id = d.pop("linkId")

        milestones = cast(list[int], d.pop("milestones"))


        name = d.pop("name")

        unclaimed = d.pop("unclaimed")

        _last_referral = d.pop("lastReferral", UNSET)
        last_referral: datetime.datetime | Unset
        if isinstance(_last_referral,  Unset):
            last_referral = UNSET
        else:
            last_referral = datetime.datetime.fromisoformat(_last_referral)




        referral_top_referrer = cls(
            counted=counted,
            email=email,
            held=held,
            in_hold=in_hold,
            link_id=link_id,
            milestones=milestones,
            name=name,
            unclaimed=unclaimed,
            last_referral=last_referral,
        )

        return referral_top_referrer

