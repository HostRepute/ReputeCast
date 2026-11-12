from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.visibility_use_email import VisibilityUseEmail





T = TypeVar("T", bound="VisibilityUsage")



@_attrs_define
class VisibilityUsage:
    """ 
        Attributes:
            blocks (int):
            emails (list[VisibilityUseEmail]):
     """

    blocks: int
    emails: list[VisibilityUseEmail]





    def to_dict(self) -> dict[str, Any]:
        from ..models.visibility_use_email import VisibilityUseEmail # noqa: PLC0415
        blocks = self.blocks

        emails = []
        for emails_item_data in self.emails:
            emails_item = emails_item_data.to_dict()
            emails.append(emails_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "blocks": blocks,
            "emails": emails,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.visibility_use_email import VisibilityUseEmail # noqa: PLC0415
        d = dict(src_dict)
        blocks = d.pop("blocks")

        emails = []
        _emails = d.pop("emails")
        for emails_item_data in (_emails):
            emails_item = VisibilityUseEmail.from_dict(emails_item_data)



            emails.append(emails_item)


        visibility_usage = cls(
            blocks=blocks,
            emails=emails,
        )

        return visibility_usage

