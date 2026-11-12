from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.preferences_page_list_text import PreferencesPageListText





T = TypeVar("T", bound="PreferencesPage")



@_attrs_define
class PreferencesPage:
    """ 
        Attributes:
            first_name (bool):
            frequency (bool):
            list_text (PreferencesPageListText):
            lists (list[str]):
            pause (bool):
     """

    first_name: bool
    frequency: bool
    list_text: PreferencesPageListText
    lists: list[str]
    pause: bool





    def to_dict(self) -> dict[str, Any]:
        from ..models.preferences_page_list_text import PreferencesPageListText # noqa: PLC0415
        first_name = self.first_name

        frequency = self.frequency

        list_text = self.list_text.to_dict()

        lists = self.lists



        pause = self.pause


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "firstName": first_name,
            "frequency": frequency,
            "listText": list_text,
            "lists": lists,
            "pause": pause,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.preferences_page_list_text import PreferencesPageListText # noqa: PLC0415
        d = dict(src_dict)
        first_name = d.pop("firstName")

        frequency = d.pop("frequency")

        list_text = PreferencesPageListText.from_dict(d.pop("listText"))




        lists = cast(list[str], d.pop("lists"))


        pause = d.pop("pause")

        preferences_page = cls(
            first_name=first_name,
            frequency=frequency,
            list_text=list_text,
            lists=lists,
            pause=pause,
        )

        return preferences_page

