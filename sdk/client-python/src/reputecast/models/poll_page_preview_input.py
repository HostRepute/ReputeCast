from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="PollPagePreviewInput")



@_attrs_define
class PollPagePreviewInput:
    """ 
        Attributes:
            config (str): The Poll block's settings, as its rc-poll attribute holds them
            campaign (str | Unset):
     """

    config: str
    campaign: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        config = self.config

        campaign = self.campaign


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "config": config,
        })
        if campaign is not UNSET:
            field_dict["campaign"] = campaign

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        config = d.pop("config")

        campaign = d.pop("campaign", UNSET)

        poll_page_preview_input = cls(
            config=config,
            campaign=campaign,
        )

        return poll_page_preview_input

