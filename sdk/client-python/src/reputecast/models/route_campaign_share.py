from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.route_campaign_share_color import RouteCampaignShareColor
from ..models.route_campaign_share_policy import RouteCampaignSharePolicy
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.route_campaign_down import RouteCampaignDown





T = TypeVar("T", bound="RouteCampaignShare")



@_attrs_define
class RouteCampaignShare:
    """ 
        Attributes:
            color (RouteCampaignShareColor):
            count (int):
            policy (RouteCampaignSharePolicy):
            provider (str):
            provider_id (str):
            short (str):
            down (RouteCampaignDown | Unset):
            fallback (str | Unset):
     """

    color: RouteCampaignShareColor
    count: int
    policy: RouteCampaignSharePolicy
    provider: str
    provider_id: str
    short: str
    down: RouteCampaignDown | Unset = UNSET
    fallback: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.route_campaign_down import RouteCampaignDown # noqa: PLC0415
        color = self.color.value

        count = self.count

        policy = self.policy.value

        provider = self.provider

        provider_id = self.provider_id

        short = self.short

        down: dict[str, Any] | Unset = UNSET
        if not isinstance(self.down, Unset):
            down = self.down.to_dict()

        fallback = self.fallback


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "color": color,
            "count": count,
            "policy": policy,
            "provider": provider,
            "providerId": provider_id,
            "short": short,
        })
        if down is not UNSET:
            field_dict["down"] = down
        if fallback is not UNSET:
            field_dict["fallback"] = fallback

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.route_campaign_down import RouteCampaignDown # noqa: PLC0415
        d = dict(src_dict)
        color = RouteCampaignShareColor(d.pop("color"))




        count = d.pop("count")

        policy = RouteCampaignSharePolicy(d.pop("policy"))




        provider = d.pop("provider")

        provider_id = d.pop("providerId")

        short = d.pop("short")

        _down = d.pop("down", UNSET)
        down: RouteCampaignDown | Unset
        if isinstance(_down,  Unset):
            down = UNSET
        else:
            down = RouteCampaignDown.from_dict(_down)




        fallback = d.pop("fallback", UNSET)

        route_campaign_share = cls(
            color=color,
            count=count,
            policy=policy,
            provider=provider,
            provider_id=provider_id,
            short=short,
            down=down,
            fallback=fallback,
        )

        return route_campaign_share

