from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.website_publish_past_input_visibility import WebsitePublishPastInputVisibility
from typing import cast






T = TypeVar("T", bound="WebsitePublishPastInput")



@_attrs_define
class WebsitePublishPastInput:
    """ 
        Attributes:
            campaigns (list[str]):
            visibility (WebsitePublishPastInputVisibility):
     """

    campaigns: list[str]
    visibility: WebsitePublishPastInputVisibility





    def to_dict(self) -> dict[str, Any]:
        campaigns = self.campaigns



        visibility = self.visibility.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "campaigns": campaigns,
            "visibility": visibility,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        campaigns = cast(list[str], d.pop("campaigns"))


        visibility = WebsitePublishPastInputVisibility(d.pop("visibility"))




        website_publish_past_input = cls(
            campaigns=campaigns,
            visibility=visibility,
        )

        return website_publish_past_input

