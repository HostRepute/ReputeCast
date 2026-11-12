from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.visibility_broken import VisibilityBroken
  from ..models.visibility_rule import VisibilityRule





T = TypeVar("T", bound="VisibilityBlockInfo")



@_attrs_define
class VisibilityBlockInfo:
    """ 
        Attributes:
            broken (list[VisibilityBroken]):
            id (str):
            name (str):
            path (list[int]):
            rule (VisibilityRule):
            section (bool):
            shown (int):
            tag (str):
            versions (list[int]):
            words (str):
     """

    broken: list[VisibilityBroken]
    id: str
    name: str
    path: list[int]
    rule: VisibilityRule
    section: bool
    shown: int
    tag: str
    versions: list[int]
    words: str





    def to_dict(self) -> dict[str, Any]:
        from ..models.visibility_broken import VisibilityBroken # noqa: PLC0415
        from ..models.visibility_rule import VisibilityRule # noqa: PLC0415
        broken = []
        for broken_item_data in self.broken:
            broken_item = broken_item_data.to_dict()
            broken.append(broken_item)



        id = self.id

        name = self.name

        path = self.path



        rule = self.rule.to_dict()

        section = self.section

        shown = self.shown

        tag = self.tag

        versions = self.versions



        words = self.words


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "broken": broken,
            "id": id,
            "name": name,
            "path": path,
            "rule": rule,
            "section": section,
            "shown": shown,
            "tag": tag,
            "versions": versions,
            "words": words,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.visibility_broken import VisibilityBroken # noqa: PLC0415
        from ..models.visibility_rule import VisibilityRule # noqa: PLC0415
        d = dict(src_dict)
        broken = []
        _broken = d.pop("broken")
        for broken_item_data in (_broken):
            broken_item = VisibilityBroken.from_dict(broken_item_data)



            broken.append(broken_item)


        id = d.pop("id")

        name = d.pop("name")

        path = cast(list[int], d.pop("path"))


        rule = VisibilityRule.from_dict(d.pop("rule"))




        section = d.pop("section")

        shown = d.pop("shown")

        tag = d.pop("tag")

        versions = cast(list[int], d.pop("versions"))


        words = d.pop("words")

        visibility_block_info = cls(
            broken=broken,
            id=id,
            name=name,
            path=path,
            rule=rule,
            section=section,
            shown=shown,
            tag=tag,
            versions=versions,
            words=words,
        )

        return visibility_block_info

