from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.upgrade_page_default_period import UpgradePageDefaultPeriod
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.upgrade_faq import UpgradeFAQ
  from ..models.upgrade_page_buttons import UpgradePageButtons





T = TypeVar("T", bound="UpgradePage")



@_attrs_define
class UpgradePage:
    """ 
        Attributes:
            default_period (UpgradePageDefaultPeriod):
            faq (list[UpgradeFAQ]):
            free_line (str):
            free_list (str):
            free_text (str):
            headline (str):
            highlight (str):
            highlight_label (str):
            intro (str):
            off (list[str]):
            order (list[str]):
            welcome (str): The welcome page after checkout; {{ first_name }} and {{ next_issue }} are filled in.
            buttons (UpgradePageButtons | Unset): A tier's button when it has no trial ("Become a founding member"), by tier
                id.
     """

    default_period: UpgradePageDefaultPeriod
    faq: list[UpgradeFAQ]
    free_line: str
    free_list: str
    free_text: str
    headline: str
    highlight: str
    highlight_label: str
    intro: str
    off: list[str]
    order: list[str]
    welcome: str
    buttons: UpgradePageButtons | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.upgrade_faq import UpgradeFAQ # noqa: PLC0415
        from ..models.upgrade_page_buttons import UpgradePageButtons # noqa: PLC0415
        default_period = self.default_period.value

        faq = []
        for faq_item_data in self.faq:
            faq_item = faq_item_data.to_dict()
            faq.append(faq_item)



        free_line = self.free_line

        free_list = self.free_list

        free_text = self.free_text

        headline = self.headline

        highlight = self.highlight

        highlight_label = self.highlight_label

        intro = self.intro

        off = self.off



        order = self.order



        welcome = self.welcome

        buttons: dict[str, Any] | Unset = UNSET
        if not isinstance(self.buttons, Unset):
            buttons = self.buttons.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "defaultPeriod": default_period,
            "faq": faq,
            "freeLine": free_line,
            "freeList": free_list,
            "freeText": free_text,
            "headline": headline,
            "highlight": highlight,
            "highlightLabel": highlight_label,
            "intro": intro,
            "off": off,
            "order": order,
            "welcome": welcome,
        })
        if buttons is not UNSET:
            field_dict["buttons"] = buttons

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.upgrade_faq import UpgradeFAQ # noqa: PLC0415
        from ..models.upgrade_page_buttons import UpgradePageButtons # noqa: PLC0415
        d = dict(src_dict)
        default_period = UpgradePageDefaultPeriod(d.pop("defaultPeriod"))




        faq = []
        _faq = d.pop("faq")
        for faq_item_data in (_faq):
            faq_item = UpgradeFAQ.from_dict(faq_item_data)



            faq.append(faq_item)


        free_line = d.pop("freeLine")

        free_list = d.pop("freeList")

        free_text = d.pop("freeText")

        headline = d.pop("headline")

        highlight = d.pop("highlight")

        highlight_label = d.pop("highlightLabel")

        intro = d.pop("intro")

        off = cast(list[str], d.pop("off"))


        order = cast(list[str], d.pop("order"))


        welcome = d.pop("welcome")

        _buttons = d.pop("buttons", UNSET)
        buttons: UpgradePageButtons | Unset
        if isinstance(_buttons,  Unset):
            buttons = UNSET
        else:
            buttons = UpgradePageButtons.from_dict(_buttons)




        upgrade_page = cls(
            default_period=default_period,
            faq=faq,
            free_line=free_line,
            free_list=free_list,
            free_text=free_text,
            headline=headline,
            highlight=highlight,
            highlight_label=highlight_label,
            intro=intro,
            off=off,
            order=order,
            welcome=welcome,
            buttons=buttons,
        )

        return upgrade_page

