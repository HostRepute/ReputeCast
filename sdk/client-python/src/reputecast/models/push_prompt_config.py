from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.push_prompt_config_corner import PushPromptConfigCorner
from ..models.push_prompt_config_ios import PushPromptConfigIos
from ..models.push_prompt_config_pages import PushPromptConfigPages
from ..models.push_prompt_config_style import PushPromptConfigStyle
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.push_prompt_when import PushPromptWhen





T = TypeVar("T", bound="PushPromptConfig")



@_attrs_define
class PushPromptConfig:
    """ 
        Attributes:
            allow (str):
            ask_again_days (int):
            ios (PushPromptConfigIos):
            message (str):
            not_now (str):
            pages (PushPromptConfigPages):
            paths (list[str]):
            style (PushPromptConfigStyle):
            title (str):
            when (PushPromptWhen):
            corner (PushPromptConfigCorner | Unset):
     """

    allow: str
    ask_again_days: int
    ios: PushPromptConfigIos
    message: str
    not_now: str
    pages: PushPromptConfigPages
    paths: list[str]
    style: PushPromptConfigStyle
    title: str
    when: PushPromptWhen
    corner: PushPromptConfigCorner | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.push_prompt_when import PushPromptWhen # noqa: PLC0415
        allow = self.allow

        ask_again_days = self.ask_again_days

        ios = self.ios.value

        message = self.message

        not_now = self.not_now

        pages = self.pages.value

        paths = self.paths



        style = self.style.value

        title = self.title

        when = self.when.to_dict()

        corner: str | Unset = UNSET
        if not isinstance(self.corner, Unset):
            corner = self.corner.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "allow": allow,
            "askAgainDays": ask_again_days,
            "ios": ios,
            "message": message,
            "notNow": not_now,
            "pages": pages,
            "paths": paths,
            "style": style,
            "title": title,
            "when": when,
        })
        if corner is not UNSET:
            field_dict["corner"] = corner

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.push_prompt_when import PushPromptWhen # noqa: PLC0415
        d = dict(src_dict)
        allow = d.pop("allow")

        ask_again_days = d.pop("askAgainDays")

        ios = PushPromptConfigIos(d.pop("ios"))




        message = d.pop("message")

        not_now = d.pop("notNow")

        pages = PushPromptConfigPages(d.pop("pages"))




        paths = cast(list[str], d.pop("paths"))


        style = PushPromptConfigStyle(d.pop("style"))




        title = d.pop("title")

        when = PushPromptWhen.from_dict(d.pop("when"))




        _corner = d.pop("corner", UNSET)
        corner: PushPromptConfigCorner | Unset
        if isinstance(_corner,  Unset):
            corner = UNSET
        else:
            corner = PushPromptConfigCorner(_corner)




        push_prompt_config = cls(
            allow=allow,
            ask_again_days=ask_again_days,
            ios=ios,
            message=message,
            not_now=not_now,
            pages=pages,
            paths=paths,
            style=style,
            title=title,
            when=when,
            corner=corner,
        )

        return push_prompt_config

