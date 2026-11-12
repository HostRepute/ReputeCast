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
  from ..models.grow_helper_sets import GrowHelperSets
  from ..models.referral_item import ReferralItem
  from ..models.referral_person import ReferralPerson
  from ..models.referral_reward_item import ReferralRewardItem
  from ..models.referral_track import ReferralTrack





T = TypeVar("T", bound="ReferralReferrerPanel")



@_attrs_define
class ReferralReferrerPanel:
    """ 
        Attributes:
            code (str):
            helpers (GrowHelperSets):
            link (str):
            link_created (datetime.datetime):
            pause_note (str):
            paused_by (str):
            paused_not_counted (int):
            paused_signals (list[str]):
            referrals (list[ReferralItem]):
            referrer (ReferralPerson):
            rewards (list[ReferralRewardItem]):
            show_addresses (bool):
            show_leaderboard (bool):
            show_name (bool):
            signups (int):
            track (ReferralTrack):
            visits (int):
            paused_at (datetime.datetime | Unset):
     """

    code: str
    helpers: GrowHelperSets
    link: str
    link_created: datetime.datetime
    pause_note: str
    paused_by: str
    paused_not_counted: int
    paused_signals: list[str]
    referrals: list[ReferralItem]
    referrer: ReferralPerson
    rewards: list[ReferralRewardItem]
    show_addresses: bool
    show_leaderboard: bool
    show_name: bool
    signups: int
    track: ReferralTrack
    visits: int
    paused_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.grow_helper_sets import GrowHelperSets # noqa: PLC0415
        from ..models.referral_item import ReferralItem # noqa: PLC0415
        from ..models.referral_person import ReferralPerson # noqa: PLC0415
        from ..models.referral_reward_item import ReferralRewardItem # noqa: PLC0415
        from ..models.referral_track import ReferralTrack # noqa: PLC0415
        code = self.code

        helpers = self.helpers.to_dict()

        link = self.link

        link_created = self.link_created.isoformat()

        pause_note = self.pause_note

        paused_by = self.paused_by

        paused_not_counted = self.paused_not_counted

        paused_signals = self.paused_signals



        referrals = []
        for referrals_item_data in self.referrals:
            referrals_item = referrals_item_data.to_dict()
            referrals.append(referrals_item)



        referrer = self.referrer.to_dict()

        rewards = []
        for rewards_item_data in self.rewards:
            rewards_item = rewards_item_data.to_dict()
            rewards.append(rewards_item)



        show_addresses = self.show_addresses

        show_leaderboard = self.show_leaderboard

        show_name = self.show_name

        signups = self.signups

        track = self.track.to_dict()

        visits = self.visits

        paused_at: str | Unset = UNSET
        if not isinstance(self.paused_at, Unset):
            paused_at = self.paused_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "code": code,
            "helpers": helpers,
            "link": link,
            "linkCreated": link_created,
            "pauseNote": pause_note,
            "pausedBy": paused_by,
            "pausedNotCounted": paused_not_counted,
            "pausedSignals": paused_signals,
            "referrals": referrals,
            "referrer": referrer,
            "rewards": rewards,
            "showAddresses": show_addresses,
            "showLeaderboard": show_leaderboard,
            "showName": show_name,
            "signups": signups,
            "track": track,
            "visits": visits,
        })
        if paused_at is not UNSET:
            field_dict["pausedAt"] = paused_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.grow_helper_sets import GrowHelperSets # noqa: PLC0415
        from ..models.referral_item import ReferralItem # noqa: PLC0415
        from ..models.referral_person import ReferralPerson # noqa: PLC0415
        from ..models.referral_reward_item import ReferralRewardItem # noqa: PLC0415
        from ..models.referral_track import ReferralTrack # noqa: PLC0415
        d = dict(src_dict)
        code = d.pop("code")

        helpers = GrowHelperSets.from_dict(d.pop("helpers"))




        link = d.pop("link")

        link_created = datetime.datetime.fromisoformat(d.pop("linkCreated"))




        pause_note = d.pop("pauseNote")

        paused_by = d.pop("pausedBy")

        paused_not_counted = d.pop("pausedNotCounted")

        paused_signals = cast(list[str], d.pop("pausedSignals"))


        referrals = []
        _referrals = d.pop("referrals")
        for referrals_item_data in (_referrals):
            referrals_item = ReferralItem.from_dict(referrals_item_data)



            referrals.append(referrals_item)


        referrer = ReferralPerson.from_dict(d.pop("referrer"))




        rewards = []
        _rewards = d.pop("rewards")
        for rewards_item_data in (_rewards):
            rewards_item = ReferralRewardItem.from_dict(rewards_item_data)



            rewards.append(rewards_item)


        show_addresses = d.pop("showAddresses")

        show_leaderboard = d.pop("showLeaderboard")

        show_name = d.pop("showName")

        signups = d.pop("signups")

        track = ReferralTrack.from_dict(d.pop("track"))




        visits = d.pop("visits")

        _paused_at = d.pop("pausedAt", UNSET)
        paused_at: datetime.datetime | Unset
        if isinstance(_paused_at,  Unset):
            paused_at = UNSET
        else:
            paused_at = datetime.datetime.fromisoformat(_paused_at)




        referral_referrer_panel = cls(
            code=code,
            helpers=helpers,
            link=link,
            link_created=link_created,
            pause_note=pause_note,
            paused_by=paused_by,
            paused_not_counted=paused_not_counted,
            paused_signals=paused_signals,
            referrals=referrals,
            referrer=referrer,
            rewards=rewards,
            show_addresses=show_addresses,
            show_leaderboard=show_leaderboard,
            show_name=show_name,
            signups=signups,
            track=track,
            visits=visits,
            paused_at=paused_at,
        )

        return referral_referrer_panel

