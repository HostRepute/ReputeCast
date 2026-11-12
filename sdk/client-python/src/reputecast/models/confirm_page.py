from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.confirm_page_list_text import ConfirmPageListText
  from ..models.confirm_page_offer_text import ConfirmPageOfferText





T = TypeVar("T", bound="ConfirmPage")



@_attrs_define
class ConfirmPage:
    """ 
        Attributes:
            inbox_tip (bool):
            list_text (ConfirmPageListText):
            offer_text (ConfirmPageOfferText):
     """

    inbox_tip: bool
    list_text: ConfirmPageListText
    offer_text: ConfirmPageOfferText





    def to_dict(self) -> dict[str, Any]:
        from ..models.confirm_page_list_text import ConfirmPageListText # noqa: PLC0415
        from ..models.confirm_page_offer_text import ConfirmPageOfferText # noqa: PLC0415
        inbox_tip = self.inbox_tip

        list_text = self.list_text.to_dict()

        offer_text = self.offer_text.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "inboxTip": inbox_tip,
            "listText": list_text,
            "offerText": offer_text,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.confirm_page_list_text import ConfirmPageListText # noqa: PLC0415
        from ..models.confirm_page_offer_text import ConfirmPageOfferText # noqa: PLC0415
        d = dict(src_dict)
        inbox_tip = d.pop("inboxTip")

        list_text = ConfirmPageListText.from_dict(d.pop("listText"))




        offer_text = ConfirmPageOfferText.from_dict(d.pop("offerText"))




        confirm_page = cls(
            inbox_tip=inbox_tip,
            list_text=list_text,
            offer_text=offer_text,
        )

        return confirm_page

