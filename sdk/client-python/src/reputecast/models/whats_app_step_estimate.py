from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.whats_app_business import WhatsAppBusiness
  from ..models.whats_app_cost import WhatsAppCost
  from ..models.whats_app_field import WhatsAppField
  from ..models.whats_app_preview import WhatsAppPreview
  from ..models.whats_app_template import WhatsAppTemplate
  from ..models.whats_app_window_example import WhatsAppWindowExample





T = TypeVar("T", bound="WhatsAppStepEstimate")



@_attrs_define
class WhatsAppStepEstimate:
    """ 
        Attributes:
            connected (bool):
            consent (int):
            cost (WhatsAppCost):
            fields (list[WhatsAppField]):
            open_ (int):
            quiet (bool):
            templates (list[WhatsAppTemplate]):
            window (str):
            business (WhatsAppBusiness | Unset):
            preview (WhatsAppPreview | Unset):
            window_example (WhatsAppWindowExample | Unset):
     """

    connected: bool
    consent: int
    cost: WhatsAppCost
    fields: list[WhatsAppField]
    open_: int
    quiet: bool
    templates: list[WhatsAppTemplate]
    window: str
    business: WhatsAppBusiness | Unset = UNSET
    preview: WhatsAppPreview | Unset = UNSET
    window_example: WhatsAppWindowExample | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.whats_app_business import WhatsAppBusiness # noqa: PLC0415
        from ..models.whats_app_cost import WhatsAppCost # noqa: PLC0415
        from ..models.whats_app_field import WhatsAppField # noqa: PLC0415
        from ..models.whats_app_preview import WhatsAppPreview # noqa: PLC0415
        from ..models.whats_app_template import WhatsAppTemplate # noqa: PLC0415
        from ..models.whats_app_window_example import WhatsAppWindowExample # noqa: PLC0415
        connected = self.connected

        consent = self.consent

        cost = self.cost.to_dict()

        fields = []
        for fields_item_data in self.fields:
            fields_item = fields_item_data.to_dict()
            fields.append(fields_item)



        open_ = self.open_

        quiet = self.quiet

        templates = []
        for templates_item_data in self.templates:
            templates_item = templates_item_data.to_dict()
            templates.append(templates_item)



        window = self.window

        business: dict[str, Any] | Unset = UNSET
        if not isinstance(self.business, Unset):
            business = self.business.to_dict()

        preview: dict[str, Any] | Unset = UNSET
        if not isinstance(self.preview, Unset):
            preview = self.preview.to_dict()

        window_example: dict[str, Any] | Unset = UNSET
        if not isinstance(self.window_example, Unset):
            window_example = self.window_example.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "connected": connected,
            "consent": consent,
            "cost": cost,
            "fields": fields,
            "open": open_,
            "quiet": quiet,
            "templates": templates,
            "window": window,
        })
        if business is not UNSET:
            field_dict["business"] = business
        if preview is not UNSET:
            field_dict["preview"] = preview
        if window_example is not UNSET:
            field_dict["windowExample"] = window_example

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.whats_app_business import WhatsAppBusiness # noqa: PLC0415
        from ..models.whats_app_cost import WhatsAppCost # noqa: PLC0415
        from ..models.whats_app_field import WhatsAppField # noqa: PLC0415
        from ..models.whats_app_preview import WhatsAppPreview # noqa: PLC0415
        from ..models.whats_app_template import WhatsAppTemplate # noqa: PLC0415
        from ..models.whats_app_window_example import WhatsAppWindowExample # noqa: PLC0415
        d = dict(src_dict)
        connected = d.pop("connected")

        consent = d.pop("consent")

        cost = WhatsAppCost.from_dict(d.pop("cost"))




        fields = []
        _fields = d.pop("fields")
        for fields_item_data in (_fields):
            fields_item = WhatsAppField.from_dict(fields_item_data)



            fields.append(fields_item)


        open_ = d.pop("open")

        quiet = d.pop("quiet")

        templates = []
        _templates = d.pop("templates")
        for templates_item_data in (_templates):
            templates_item = WhatsAppTemplate.from_dict(templates_item_data)



            templates.append(templates_item)


        window = d.pop("window")

        _business = d.pop("business", UNSET)
        business: WhatsAppBusiness | Unset
        if isinstance(_business,  Unset):
            business = UNSET
        else:
            business = WhatsAppBusiness.from_dict(_business)




        _preview = d.pop("preview", UNSET)
        preview: WhatsAppPreview | Unset
        if isinstance(_preview,  Unset):
            preview = UNSET
        else:
            preview = WhatsAppPreview.from_dict(_preview)




        _window_example = d.pop("windowExample", UNSET)
        window_example: WhatsAppWindowExample | Unset
        if isinstance(_window_example,  Unset):
            window_example = UNSET
        else:
            window_example = WhatsAppWindowExample.from_dict(_window_example)




        whats_app_step_estimate = cls(
            connected=connected,
            consent=consent,
            cost=cost,
            fields=fields,
            open_=open_,
            quiet=quiet,
            templates=templates,
            window=window,
            business=business,
            preview=preview,
            window_example=window_example,
        )

        return whats_app_step_estimate

