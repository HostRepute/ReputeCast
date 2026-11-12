from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.grow_reward_count import GrowRewardCount





T = TypeVar("T", bound="GrowOverviewTiles")



@_attrs_define
class GrowOverviewTiles:
    """ 
        Attributes:
            confirmed (int):
            conversion (float): Percent
            from_grow (int):
            new_subscribers (int):
            partner_signups (int):
            partners (list[str]):
            rewards (list[GrowRewardCount]):
            rewards_sent (int):
            visits (int):
     """

    confirmed: int
    conversion: float
    from_grow: int
    new_subscribers: int
    partner_signups: int
    partners: list[str]
    rewards: list[GrowRewardCount]
    rewards_sent: int
    visits: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.grow_reward_count import GrowRewardCount # noqa: PLC0415
        confirmed = self.confirmed

        conversion = self.conversion

        from_grow = self.from_grow

        new_subscribers = self.new_subscribers

        partner_signups = self.partner_signups

        partners = self.partners



        rewards = []
        for rewards_item_data in self.rewards:
            rewards_item = rewards_item_data.to_dict()
            rewards.append(rewards_item)



        rewards_sent = self.rewards_sent

        visits = self.visits


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "confirmed": confirmed,
            "conversion": conversion,
            "fromGrow": from_grow,
            "newSubscribers": new_subscribers,
            "partnerSignups": partner_signups,
            "partners": partners,
            "rewards": rewards,
            "rewardsSent": rewards_sent,
            "visits": visits,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.grow_reward_count import GrowRewardCount # noqa: PLC0415
        d = dict(src_dict)
        confirmed = d.pop("confirmed")

        conversion = d.pop("conversion")

        from_grow = d.pop("fromGrow")

        new_subscribers = d.pop("newSubscribers")

        partner_signups = d.pop("partnerSignups")

        partners = cast(list[str], d.pop("partners"))


        rewards = []
        _rewards = d.pop("rewards")
        for rewards_item_data in (_rewards):
            rewards_item = GrowRewardCount.from_dict(rewards_item_data)



            rewards.append(rewards_item)


        rewards_sent = d.pop("rewardsSent")

        visits = d.pop("visits")

        grow_overview_tiles = cls(
            confirmed=confirmed,
            conversion=conversion,
            from_grow=from_grow,
            new_subscribers=new_subscribers,
            partner_signups=partner_signups,
            partners=partners,
            rewards=rewards,
            rewards_sent=rewards_sent,
            visits=visits,
        )

        return grow_overview_tiles

