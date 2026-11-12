from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.earn_preview_input_as import EarnPreviewInputAs
from typing import cast

if TYPE_CHECKING:
  from ..models.upgrade_page import UpgradePage





T = TypeVar("T", bound="EarnPreviewInput")



@_attrs_define
class EarnPreviewInput:
    """ 
        Attributes:
            as_ (EarnPreviewInputAs):  Default: EarnPreviewInputAs.VISITOR.
            page (UpgradePage):
     """

    page: UpgradePage
    as_: EarnPreviewInputAs = EarnPreviewInputAs.VISITOR





    def to_dict(self) -> dict[str, Any]:
        from ..models.upgrade_page import UpgradePage # noqa: PLC0415
        as_ = self.as_.value

        page = self.page.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "as": as_,
            "page": page,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.upgrade_page import UpgradePage # noqa: PLC0415
        d = dict(src_dict)
        as_ = EarnPreviewInputAs(d.pop("as"))




        page = UpgradePage.from_dict(d.pop("page"))




        earn_preview_input = cls(
            as_=as_,
            page=page,
        )

        return earn_preview_input

