from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.referral_page_settings_share_buttons_item import ReferralPageSettingsShareButtonsItem
from typing import cast






T = TypeVar("T", bound="ReferralPageSettings")



@_attrs_define
class ReferralPageSettings:
    """ 
        Attributes:
            hub_intro (str):
            invite_headline (str):
            invite_text (str):
            share_buttons (list[ReferralPageSettingsShareButtonsItem]):
            share_message (str):
     """

    hub_intro: str
    invite_headline: str
    invite_text: str
    share_buttons: list[ReferralPageSettingsShareButtonsItem]
    share_message: str





    def to_dict(self) -> dict[str, Any]:
        hub_intro = self.hub_intro

        invite_headline = self.invite_headline

        invite_text = self.invite_text

        share_buttons = []
        for share_buttons_item_data in self.share_buttons:
            share_buttons_item = share_buttons_item_data.value
            share_buttons.append(share_buttons_item)



        share_message = self.share_message


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "hubIntro": hub_intro,
            "inviteHeadline": invite_headline,
            "inviteText": invite_text,
            "shareButtons": share_buttons,
            "shareMessage": share_message,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        hub_intro = d.pop("hubIntro")

        invite_headline = d.pop("inviteHeadline")

        invite_text = d.pop("inviteText")

        share_buttons = []
        _share_buttons = d.pop("shareButtons")
        for share_buttons_item_data in (_share_buttons):
            share_buttons_item = ReferralPageSettingsShareButtonsItem(share_buttons_item_data)



            share_buttons.append(share_buttons_item)


        share_message = d.pop("shareMessage")

        referral_page_settings = cls(
            hub_intro=hub_intro,
            invite_headline=invite_headline,
            invite_text=invite_text,
            share_buttons=share_buttons,
            share_message=share_message,
        )

        return referral_page_settings

