from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.recommendations_block_layout import RecommendationsBlockLayout
from ..models.recommendations_block_which import RecommendationsBlockWhich
from typing import cast






T = TypeVar("T", bound="RecommendationsBlock")



@_attrs_define
class RecommendationsBlock:
    """ 
        Attributes:
            heading (str):
            ids (list[str]):
            intro (str):
            layout (RecommendationsBlockLayout):
            which (RecommendationsBlockWhich):
     """

    heading: str
    ids: list[str]
    intro: str
    layout: RecommendationsBlockLayout
    which: RecommendationsBlockWhich





    def to_dict(self) -> dict[str, Any]:
        heading = self.heading

        ids = self.ids



        intro = self.intro

        layout = self.layout.value

        which = self.which.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "heading": heading,
            "ids": ids,
            "intro": intro,
            "layout": layout,
            "which": which,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        heading = d.pop("heading")

        ids = cast(list[str], d.pop("ids"))


        intro = d.pop("intro")

        layout = RecommendationsBlockLayout(d.pop("layout"))




        which = RecommendationsBlockWhich(d.pop("which"))




        recommendations_block = cls(
            heading=heading,
            ids=ids,
            intro=intro,
            layout=layout,
            which=which,
        )

        return recommendations_block

