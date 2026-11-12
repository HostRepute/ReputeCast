from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.referral_contact_referrer_link_status import ReferralContactReferrerLinkStatus
from typing import cast

if TYPE_CHECKING:
  from ..models.referral_contact_reward import ReferralContactReward





T = TypeVar("T", bound="ReferralContactReferrer")



@_attrs_define
class ReferralContactReferrer:
    """ 
        Attributes:
            awaiting (int):
            code (str):
            counted (int):
            held (int):
            in_hold (int):
            link (str):
            link_id (str):
            link_status (ReferralContactReferrerLinkStatus):
            list_name (str):
            milestones (list[int]):
            program_id (str):
            rewards (list[ReferralContactReward]):
            signups (int):
            visits (int):
     """

    awaiting: int
    code: str
    counted: int
    held: int
    in_hold: int
    link: str
    link_id: str
    link_status: ReferralContactReferrerLinkStatus
    list_name: str
    milestones: list[int]
    program_id: str
    rewards: list[ReferralContactReward]
    signups: int
    visits: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.referral_contact_reward import ReferralContactReward # noqa: PLC0415
        awaiting = self.awaiting

        code = self.code

        counted = self.counted

        held = self.held

        in_hold = self.in_hold

        link = self.link

        link_id = self.link_id

        link_status = self.link_status.value

        list_name = self.list_name

        milestones = self.milestones



        program_id = self.program_id

        rewards = []
        for rewards_item_data in self.rewards:
            rewards_item = rewards_item_data.to_dict()
            rewards.append(rewards_item)



        signups = self.signups

        visits = self.visits


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "awaiting": awaiting,
            "code": code,
            "counted": counted,
            "held": held,
            "inHold": in_hold,
            "link": link,
            "linkId": link_id,
            "linkStatus": link_status,
            "listName": list_name,
            "milestones": milestones,
            "programId": program_id,
            "rewards": rewards,
            "signups": signups,
            "visits": visits,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.referral_contact_reward import ReferralContactReward # noqa: PLC0415
        d = dict(src_dict)
        awaiting = d.pop("awaiting")

        code = d.pop("code")

        counted = d.pop("counted")

        held = d.pop("held")

        in_hold = d.pop("inHold")

        link = d.pop("link")

        link_id = d.pop("linkId")

        link_status = ReferralContactReferrerLinkStatus(d.pop("linkStatus"))




        list_name = d.pop("listName")

        milestones = cast(list[int], d.pop("milestones"))


        program_id = d.pop("programId")

        rewards = []
        _rewards = d.pop("rewards")
        for rewards_item_data in (_rewards):
            rewards_item = ReferralContactReward.from_dict(rewards_item_data)



            rewards.append(rewards_item)


        signups = d.pop("signups")

        visits = d.pop("visits")

        referral_contact_referrer = cls(
            awaiting=awaiting,
            code=code,
            counted=counted,
            held=held,
            in_hold=in_hold,
            link=link,
            link_id=link_id,
            link_status=link_status,
            list_name=list_name,
            milestones=milestones,
            program_id=program_id,
            rewards=rewards,
            signups=signups,
            visits=visits,
        )

        return referral_contact_referrer

