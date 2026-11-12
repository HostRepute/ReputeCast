from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.push_prompt_rule import PushPromptRule





T = TypeVar("T", bound="PushPromptWhen")



@_attrs_define
class PushPromptWhen:
    """ 
        Attributes:
            button (bool):
            posts (PushPromptRule):
            scroll (PushPromptRule):
            seconds (PushPromptRule):
     """

    button: bool
    posts: PushPromptRule
    scroll: PushPromptRule
    seconds: PushPromptRule





    def to_dict(self) -> dict[str, Any]:
        from ..models.push_prompt_rule import PushPromptRule # noqa: PLC0415
        button = self.button

        posts = self.posts.to_dict()

        scroll = self.scroll.to_dict()

        seconds = self.seconds.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "button": button,
            "posts": posts,
            "scroll": scroll,
            "seconds": seconds,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.push_prompt_rule import PushPromptRule # noqa: PLC0415
        d = dict(src_dict)
        button = d.pop("button")

        posts = PushPromptRule.from_dict(d.pop("posts"))




        scroll = PushPromptRule.from_dict(d.pop("scroll"))




        seconds = PushPromptRule.from_dict(d.pop("seconds"))




        push_prompt_when = cls(
            button=button,
            posts=posts,
            scroll=scroll,
            seconds=seconds,
        )

        return push_prompt_when

