from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.referral_results_grouping import ReferralResultsGrouping
from ..models.referral_results_period import ReferralResultsPeriod
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.referral_bucket import ReferralBucket
  from ..models.referral_channel import ReferralChannel
  from ..models.referral_funnel import ReferralFunnel
  from ..models.referral_program_detail import ReferralProgramDetail
  from ..models.referral_quality import ReferralQuality
  from ..models.referral_reward_row import ReferralRewardRow
  from ..models.referral_top_referrer import ReferralTopReferrer





T = TypeVar("T", bound="ReferralResults")



@_attrs_define
class ReferralResults:
    """ 
        Attributes:
            channels (list[ReferralChannel]):
            enough (bool): 20 confirmed referrals or more
            from_ (datetime.datetime):
            funnel (ReferralFunnel):
            grouping (ReferralResultsGrouping):
            over_time (list[ReferralBucket]):
            period (ReferralResultsPeriod):
            program (ReferralProgramDetail):
            quality (ReferralQuality):
            rewards (list[ReferralRewardRow]):
            to (datetime.datetime):
            top (list[ReferralTopReferrer]):
            with_ten (int):
            with_three (int):
     """

    channels: list[ReferralChannel]
    enough: bool
    from_: datetime.datetime
    funnel: ReferralFunnel
    grouping: ReferralResultsGrouping
    over_time: list[ReferralBucket]
    period: ReferralResultsPeriod
    program: ReferralProgramDetail
    quality: ReferralQuality
    rewards: list[ReferralRewardRow]
    to: datetime.datetime
    top: list[ReferralTopReferrer]
    with_ten: int
    with_three: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.referral_bucket import ReferralBucket # noqa: PLC0415
        from ..models.referral_channel import ReferralChannel # noqa: PLC0415
        from ..models.referral_funnel import ReferralFunnel # noqa: PLC0415
        from ..models.referral_program_detail import ReferralProgramDetail # noqa: PLC0415
        from ..models.referral_quality import ReferralQuality # noqa: PLC0415
        from ..models.referral_reward_row import ReferralRewardRow # noqa: PLC0415
        from ..models.referral_top_referrer import ReferralTopReferrer # noqa: PLC0415
        channels = []
        for channels_item_data in self.channels:
            channels_item = channels_item_data.to_dict()
            channels.append(channels_item)



        enough = self.enough

        from_ = self.from_.isoformat()

        funnel = self.funnel.to_dict()

        grouping = self.grouping.value

        over_time = []
        for over_time_item_data in self.over_time:
            over_time_item = over_time_item_data.to_dict()
            over_time.append(over_time_item)



        period = self.period.value

        program = self.program.to_dict()

        quality = self.quality.to_dict()

        rewards = []
        for rewards_item_data in self.rewards:
            rewards_item = rewards_item_data.to_dict()
            rewards.append(rewards_item)



        to = self.to.isoformat()

        top = []
        for top_item_data in self.top:
            top_item = top_item_data.to_dict()
            top.append(top_item)



        with_ten = self.with_ten

        with_three = self.with_three


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "channels": channels,
            "enough": enough,
            "from": from_,
            "funnel": funnel,
            "grouping": grouping,
            "overTime": over_time,
            "period": period,
            "program": program,
            "quality": quality,
            "rewards": rewards,
            "to": to,
            "top": top,
            "withTen": with_ten,
            "withThree": with_three,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.referral_bucket import ReferralBucket # noqa: PLC0415
        from ..models.referral_channel import ReferralChannel # noqa: PLC0415
        from ..models.referral_funnel import ReferralFunnel # noqa: PLC0415
        from ..models.referral_program_detail import ReferralProgramDetail # noqa: PLC0415
        from ..models.referral_quality import ReferralQuality # noqa: PLC0415
        from ..models.referral_reward_row import ReferralRewardRow # noqa: PLC0415
        from ..models.referral_top_referrer import ReferralTopReferrer # noqa: PLC0415
        d = dict(src_dict)
        channels = []
        _channels = d.pop("channels")
        for channels_item_data in (_channels):
            channels_item = ReferralChannel.from_dict(channels_item_data)



            channels.append(channels_item)


        enough = d.pop("enough")

        from_ = datetime.datetime.fromisoformat(d.pop("from"))




        funnel = ReferralFunnel.from_dict(d.pop("funnel"))




        grouping = ReferralResultsGrouping(d.pop("grouping"))




        over_time = []
        _over_time = d.pop("overTime")
        for over_time_item_data in (_over_time):
            over_time_item = ReferralBucket.from_dict(over_time_item_data)



            over_time.append(over_time_item)


        period = ReferralResultsPeriod(d.pop("period"))




        program = ReferralProgramDetail.from_dict(d.pop("program"))




        quality = ReferralQuality.from_dict(d.pop("quality"))




        rewards = []
        _rewards = d.pop("rewards")
        for rewards_item_data in (_rewards):
            rewards_item = ReferralRewardRow.from_dict(rewards_item_data)



            rewards.append(rewards_item)


        to = datetime.datetime.fromisoformat(d.pop("to"))




        top = []
        _top = d.pop("top")
        for top_item_data in (_top):
            top_item = ReferralTopReferrer.from_dict(top_item_data)



            top.append(top_item)


        with_ten = d.pop("withTen")

        with_three = d.pop("withThree")

        referral_results = cls(
            channels=channels,
            enough=enough,
            from_=from_,
            funnel=funnel,
            grouping=grouping,
            over_time=over_time,
            period=period,
            program=program,
            quality=quality,
            rewards=rewards,
            to=to,
            top=top,
            with_ten=with_ten,
            with_three=with_three,
        )

        return referral_results

