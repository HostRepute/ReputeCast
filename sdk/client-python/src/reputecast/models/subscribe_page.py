from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.subscribe_page_list_text import SubscribePageListText





T = TypeVar("T", bound="SubscribePage")



@_attrs_define
class SubscribePage:
    """ 
        Attributes:
            description (str):
            footnote (str):
            heading (str):
            index (bool):
            list_text (SubscribePageListText):
     """

    description: str
    footnote: str
    heading: str
    index: bool
    list_text: SubscribePageListText





    def to_dict(self) -> dict[str, Any]:
        from ..models.subscribe_page_list_text import SubscribePageListText # noqa: PLC0415
        description = self.description

        footnote = self.footnote

        heading = self.heading

        index = self.index

        list_text = self.list_text.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "description": description,
            "footnote": footnote,
            "heading": heading,
            "index": index,
            "listText": list_text,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.subscribe_page_list_text import SubscribePageListText # noqa: PLC0415
        d = dict(src_dict)
        description = d.pop("description")

        footnote = d.pop("footnote")

        heading = d.pop("heading")

        index = d.pop("index")

        list_text = SubscribePageListText.from_dict(d.pop("listText"))




        subscribe_page = cls(
            description=description,
            footnote=footnote,
            heading=heading,
            index=index,
            list_text=list_text,
        )

        return subscribe_page

