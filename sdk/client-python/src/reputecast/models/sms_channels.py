from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.channel_status import ChannelStatus





T = TypeVar("T", bound="SmsChannels")



@_attrs_define
class SmsChannels:
    """ 
        Attributes:
            channels (list[ChannelStatus]):
            switch_shown (bool):
     """

    channels: list[ChannelStatus]
    switch_shown: bool





    def to_dict(self) -> dict[str, Any]:
        from ..models.channel_status import ChannelStatus # noqa: PLC0415
        channels = []
        for channels_item_data in self.channels:
            channels_item = channels_item_data.to_dict()
            channels.append(channels_item)



        switch_shown = self.switch_shown


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "channels": channels,
            "switchShown": switch_shown,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.channel_status import ChannelStatus # noqa: PLC0415
        d = dict(src_dict)
        channels = []
        _channels = d.pop("channels")
        for channels_item_data in (_channels):
            channels_item = ChannelStatus.from_dict(channels_item_data)



            channels.append(channels_item)


        switch_shown = d.pop("switchShown")

        sms_channels = cls(
            channels=channels,
            switch_shown=switch_shown,
        )

        return sms_channels

