from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.route_campaign_mailbox_cap import RouteCampaignMailboxCap





T = TypeVar("T", bound="RouteCampaignWarmup")



@_attrs_define
class RouteCampaignWarmup:
    """ 
        Attributes:
            cap (int):
            date (datetime.datetime):
            day (int):
            left (int):
            mailbox_caps (list[RouteCampaignMailboxCap]):
            over (int):
            paused (bool):
            provider (str):
            provider_id (str):
            spread (bool):
            tier (int):
            resumes (datetime.datetime | Unset):
            tier_to (str | Unset):
            to (str | Unset):
     """

    cap: int
    date: datetime.datetime
    day: int
    left: int
    mailbox_caps: list[RouteCampaignMailboxCap]
    over: int
    paused: bool
    provider: str
    provider_id: str
    spread: bool
    tier: int
    resumes: datetime.datetime | Unset = UNSET
    tier_to: str | Unset = UNSET
    to: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.route_campaign_mailbox_cap import RouteCampaignMailboxCap # noqa: PLC0415
        cap = self.cap

        date = self.date.isoformat()

        day = self.day

        left = self.left

        mailbox_caps = []
        for mailbox_caps_item_data in self.mailbox_caps:
            mailbox_caps_item = mailbox_caps_item_data.to_dict()
            mailbox_caps.append(mailbox_caps_item)



        over = self.over

        paused = self.paused

        provider = self.provider

        provider_id = self.provider_id

        spread = self.spread

        tier = self.tier

        resumes: str | Unset = UNSET
        if not isinstance(self.resumes, Unset):
            resumes = self.resumes.isoformat()

        tier_to = self.tier_to

        to = self.to


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "cap": cap,
            "date": date,
            "day": day,
            "left": left,
            "mailboxCaps": mailbox_caps,
            "over": over,
            "paused": paused,
            "provider": provider,
            "providerId": provider_id,
            "spread": spread,
            "tier": tier,
        })
        if resumes is not UNSET:
            field_dict["resumes"] = resumes
        if tier_to is not UNSET:
            field_dict["tierTo"] = tier_to
        if to is not UNSET:
            field_dict["to"] = to

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.route_campaign_mailbox_cap import RouteCampaignMailboxCap # noqa: PLC0415
        d = dict(src_dict)
        cap = d.pop("cap")

        date = datetime.datetime.fromisoformat(d.pop("date"))




        day = d.pop("day")

        left = d.pop("left")

        mailbox_caps = []
        _mailbox_caps = d.pop("mailboxCaps")
        for mailbox_caps_item_data in (_mailbox_caps):
            mailbox_caps_item = RouteCampaignMailboxCap.from_dict(mailbox_caps_item_data)



            mailbox_caps.append(mailbox_caps_item)


        over = d.pop("over")

        paused = d.pop("paused")

        provider = d.pop("provider")

        provider_id = d.pop("providerId")

        spread = d.pop("spread")

        tier = d.pop("tier")

        _resumes = d.pop("resumes", UNSET)
        resumes: datetime.datetime | Unset
        if isinstance(_resumes,  Unset):
            resumes = UNSET
        else:
            resumes = datetime.datetime.fromisoformat(_resumes)




        tier_to = d.pop("tierTo", UNSET)

        to = d.pop("to", UNSET)

        route_campaign_warmup = cls(
            cap=cap,
            date=date,
            day=day,
            left=left,
            mailbox_caps=mailbox_caps,
            over=over,
            paused=paused,
            provider=provider,
            provider_id=provider_id,
            spread=spread,
            tier=tier,
            resumes=resumes,
            tier_to=tier_to,
            to=to,
        )

        return route_campaign_warmup

