from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.website_post_preview_input_as import WebsitePostPreviewInputAs
from ..types import UNSET, Unset






T = TypeVar("T", bound="WebsitePostPreviewInput")



@_attrs_define
class WebsitePostPreviewInput:
    """ 
        Attributes:
            as_ (WebsitePostPreviewInputAs): A visitor (first metered read, or the meter used up), a free member (or the
                meter used up), a paid member
            tier (str | Unset): A paid member's tier (the highest by default)
     """

    as_: WebsitePostPreviewInputAs
    tier: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        as_ = self.as_.value

        tier = self.tier


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "as": as_,
        })
        if tier is not UNSET:
            field_dict["tier"] = tier

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        as_ = WebsitePostPreviewInputAs(d.pop("as"))




        tier = d.pop("tier", UNSET)

        website_post_preview_input = cls(
            as_=as_,
            tier=tier,
        )

        return website_post_preview_input

