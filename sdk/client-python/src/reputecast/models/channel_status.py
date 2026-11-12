from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.channel_status_channel import ChannelStatusChannel






T = TypeVar("T", bound="ChannelStatus")



@_attrs_define
class ChannelStatus:
    """ 
        Attributes:
            available (bool):
            channel (ChannelStatusChannel):
            set_up (bool):
     """

    available: bool
    channel: ChannelStatusChannel
    set_up: bool





    def to_dict(self) -> dict[str, Any]:
        available = self.available

        channel = self.channel.value

        set_up = self.set_up


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "available": available,
            "channel": channel,
            "setUp": set_up,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        available = d.pop("available")

        channel = ChannelStatusChannel(d.pop("channel"))




        set_up = d.pop("setUp")

        channel_status = cls(
            available=available,
            channel=channel,
            set_up=set_up,
        )

        return channel_status

