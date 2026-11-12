from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.automation_starter_template_key import AutomationStarterTemplateKey
from typing import cast






T = TypeVar("T", bound="AutomationStarterTemplate")



@_attrs_define
class AutomationStarterTemplate:
    """ 
        Attributes:
            key (AutomationStarterTemplateKey):
            name (str):
            outline (str):
            preview (list[str]):
            steps (int):
            summary (str):
     """

    key: AutomationStarterTemplateKey
    name: str
    outline: str
    preview: list[str]
    steps: int
    summary: str





    def to_dict(self) -> dict[str, Any]:
        key = self.key.value

        name = self.name

        outline = self.outline

        preview = self.preview



        steps = self.steps

        summary = self.summary


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "key": key,
            "name": name,
            "outline": outline,
            "preview": preview,
            "steps": steps,
            "summary": summary,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        key = AutomationStarterTemplateKey(d.pop("key"))




        name = d.pop("name")

        outline = d.pop("outline")

        preview = cast(list[str], d.pop("preview"))


        steps = d.pop("steps")

        summary = d.pop("summary")

        automation_starter_template = cls(
            key=key,
            name=name,
            outline=outline,
            preview=preview,
            steps=steps,
            summary=summary,
        )

        return automation_starter_template

