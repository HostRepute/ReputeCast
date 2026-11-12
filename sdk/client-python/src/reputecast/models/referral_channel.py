from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.referral_channel_channel import ReferralChannelChannel






T = TypeVar("T", bound="ReferralChannel")



@_attrs_define
class ReferralChannel:
    """ 
        Attributes:
            channel (ReferralChannelChannel):
            signups (int):
            visits (int):
     """

    channel: ReferralChannelChannel
    signups: int
    visits: int





    def to_dict(self) -> dict[str, Any]:
        channel = self.channel.value

        signups = self.signups

        visits = self.visits


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "channel": channel,
            "signups": signups,
            "visits": visits,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        channel = ReferralChannelChannel(d.pop("channel"))




        signups = d.pop("signups")

        visits = d.pop("visits")

        referral_channel = cls(
            channel=channel,
            signups=signups,
            visits=visits,
        )

        return referral_channel

