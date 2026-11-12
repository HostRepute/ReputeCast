from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.public_page_copy_input_strings import PublicPageCopyInputStrings





T = TypeVar("T", bound="PublicPageCopyInput")



@_attrs_define
class PublicPageCopyInput:
    """ 
        Attributes:
            strings (PublicPageCopyInputStrings):
     """

    strings: PublicPageCopyInputStrings





    def to_dict(self) -> dict[str, Any]:
        from ..models.public_page_copy_input_strings import PublicPageCopyInputStrings # noqa: PLC0415
        strings = self.strings.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "strings": strings,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.public_page_copy_input_strings import PublicPageCopyInputStrings # noqa: PLC0415
        d = dict(src_dict)
        strings = PublicPageCopyInputStrings.from_dict(d.pop("strings"))




        public_page_copy_input = cls(
            strings=strings,
        )

        return public_page_copy_input

