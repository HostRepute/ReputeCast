from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.whats_app_fallback import WhatsAppFallback
  from ..models.whats_app_template_button import WhatsAppTemplateButton





T = TypeVar("T", bound="WhatsAppPreview")



@_attrs_define
class WhatsAppPreview:
    """ 
        Attributes:
            buttons (list[WhatsAppTemplateButton]):
            contact (str):
            fallbacks (list[WhatsAppFallback]):
            text (str):
            contact_id (str | Unset):
            first_name (str | Unset):
     """

    buttons: list[WhatsAppTemplateButton]
    contact: str
    fallbacks: list[WhatsAppFallback]
    text: str
    contact_id: str | Unset = UNSET
    first_name: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.whats_app_fallback import WhatsAppFallback # noqa: PLC0415
        from ..models.whats_app_template_button import WhatsAppTemplateButton # noqa: PLC0415
        buttons = []
        for buttons_item_data in self.buttons:
            buttons_item = buttons_item_data.to_dict()
            buttons.append(buttons_item)



        contact = self.contact

        fallbacks = []
        for fallbacks_item_data in self.fallbacks:
            fallbacks_item = fallbacks_item_data.to_dict()
            fallbacks.append(fallbacks_item)



        text = self.text

        contact_id = self.contact_id

        first_name = self.first_name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "buttons": buttons,
            "contact": contact,
            "fallbacks": fallbacks,
            "text": text,
        })
        if contact_id is not UNSET:
            field_dict["contactId"] = contact_id
        if first_name is not UNSET:
            field_dict["firstName"] = first_name

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.whats_app_fallback import WhatsAppFallback # noqa: PLC0415
        from ..models.whats_app_template_button import WhatsAppTemplateButton # noqa: PLC0415
        d = dict(src_dict)
        buttons = []
        _buttons = d.pop("buttons")
        for buttons_item_data in (_buttons):
            buttons_item = WhatsAppTemplateButton.from_dict(buttons_item_data)



            buttons.append(buttons_item)


        contact = d.pop("contact")

        fallbacks = []
        _fallbacks = d.pop("fallbacks")
        for fallbacks_item_data in (_fallbacks):
            fallbacks_item = WhatsAppFallback.from_dict(fallbacks_item_data)



            fallbacks.append(fallbacks_item)


        text = d.pop("text")

        contact_id = d.pop("contactId", UNSET)

        first_name = d.pop("firstName", UNSET)

        whats_app_preview = cls(
            buttons=buttons,
            contact=contact,
            fallbacks=fallbacks,
            text=text,
            contact_id=contact_id,
            first_name=first_name,
        )

        return whats_app_preview

