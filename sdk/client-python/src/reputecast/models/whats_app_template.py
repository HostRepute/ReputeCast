from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.whats_app_template_category import WhatsAppTemplateCategory
from ..models.whats_app_template_status import WhatsAppTemplateStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.whats_app_template_button import WhatsAppTemplateButton
  from ..models.whats_app_template_var import WhatsAppTemplateVar





T = TypeVar("T", bound="WhatsAppTemplate")



@_attrs_define
class WhatsAppTemplate:
    """ 
        Attributes:
            body (str):
            buttons (list[WhatsAppTemplateButton]):
            category (WhatsAppTemplateCategory):
            id (str):
            language (str):
            language_label (str):
            name (str):
            status (WhatsAppTemplateStatus):
            synced_at (datetime.datetime):
            variables (list[WhatsAppTemplateVar]):
            reason (str | Unset):
            reason_code (str | Unset):
            status_since (datetime.datetime | Unset):
     """

    body: str
    buttons: list[WhatsAppTemplateButton]
    category: WhatsAppTemplateCategory
    id: str
    language: str
    language_label: str
    name: str
    status: WhatsAppTemplateStatus
    synced_at: datetime.datetime
    variables: list[WhatsAppTemplateVar]
    reason: str | Unset = UNSET
    reason_code: str | Unset = UNSET
    status_since: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.whats_app_template_button import WhatsAppTemplateButton # noqa: PLC0415
        from ..models.whats_app_template_var import WhatsAppTemplateVar # noqa: PLC0415
        body = self.body

        buttons = []
        for buttons_item_data in self.buttons:
            buttons_item = buttons_item_data.to_dict()
            buttons.append(buttons_item)



        category = self.category.value

        id = self.id

        language = self.language

        language_label = self.language_label

        name = self.name

        status = self.status.value

        synced_at = self.synced_at.isoformat()

        variables = []
        for variables_item_data in self.variables:
            variables_item = variables_item_data.to_dict()
            variables.append(variables_item)



        reason = self.reason

        reason_code = self.reason_code

        status_since: str | Unset = UNSET
        if not isinstance(self.status_since, Unset):
            status_since = self.status_since.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "body": body,
            "buttons": buttons,
            "category": category,
            "id": id,
            "language": language,
            "languageLabel": language_label,
            "name": name,
            "status": status,
            "syncedAt": synced_at,
            "variables": variables,
        })
        if reason is not UNSET:
            field_dict["reason"] = reason
        if reason_code is not UNSET:
            field_dict["reasonCode"] = reason_code
        if status_since is not UNSET:
            field_dict["statusSince"] = status_since

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.whats_app_template_button import WhatsAppTemplateButton # noqa: PLC0415
        from ..models.whats_app_template_var import WhatsAppTemplateVar # noqa: PLC0415
        d = dict(src_dict)
        body = d.pop("body")

        buttons = []
        _buttons = d.pop("buttons")
        for buttons_item_data in (_buttons):
            buttons_item = WhatsAppTemplateButton.from_dict(buttons_item_data)



            buttons.append(buttons_item)


        category = WhatsAppTemplateCategory(d.pop("category"))




        id = d.pop("id")

        language = d.pop("language")

        language_label = d.pop("languageLabel")

        name = d.pop("name")

        status = WhatsAppTemplateStatus(d.pop("status"))




        synced_at = datetime.datetime.fromisoformat(d.pop("syncedAt"))




        variables = []
        _variables = d.pop("variables")
        for variables_item_data in (_variables):
            variables_item = WhatsAppTemplateVar.from_dict(variables_item_data)



            variables.append(variables_item)


        reason = d.pop("reason", UNSET)

        reason_code = d.pop("reasonCode", UNSET)

        _status_since = d.pop("statusSince", UNSET)
        status_since: datetime.datetime | Unset
        if isinstance(_status_since,  Unset):
            status_since = UNSET
        else:
            status_since = datetime.datetime.fromisoformat(_status_since)




        whats_app_template = cls(
            body=body,
            buttons=buttons,
            category=category,
            id=id,
            language=language,
            language_label=language_label,
            name=name,
            status=status,
            synced_at=synced_at,
            variables=variables,
            reason=reason,
            reason_code=reason_code,
            status_since=status_since,
        )

        return whats_app_template

