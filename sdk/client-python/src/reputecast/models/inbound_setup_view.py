from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.inbound_mx_record import InboundMXRecord
  from ..models.inbound_setup_field import InboundSetupField





T = TypeVar("T", bound="InboundSetupView")



@_attrs_define
class InboundSetupView:
    """ 
        Attributes:
            automatic (bool):
            mx (list[InboundMXRecord]):
            settings (list[InboundSetupField]):
            steps (list[str]):
     """

    automatic: bool
    mx: list[InboundMXRecord]
    settings: list[InboundSetupField]
    steps: list[str]





    def to_dict(self) -> dict[str, Any]:
        from ..models.inbound_mx_record import InboundMXRecord # noqa: PLC0415
        from ..models.inbound_setup_field import InboundSetupField # noqa: PLC0415
        automatic = self.automatic

        mx = []
        for mx_item_data in self.mx:
            mx_item = mx_item_data.to_dict()
            mx.append(mx_item)



        settings = []
        for settings_item_data in self.settings:
            settings_item = settings_item_data.to_dict()
            settings.append(settings_item)



        steps = self.steps




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "automatic": automatic,
            "mx": mx,
            "settings": settings,
            "steps": steps,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.inbound_mx_record import InboundMXRecord # noqa: PLC0415
        from ..models.inbound_setup_field import InboundSetupField # noqa: PLC0415
        d = dict(src_dict)
        automatic = d.pop("automatic")

        mx = []
        _mx = d.pop("mx")
        for mx_item_data in (_mx):
            mx_item = InboundMXRecord.from_dict(mx_item_data)



            mx.append(mx_item)


        settings = []
        _settings = d.pop("settings")
        for settings_item_data in (_settings):
            settings_item = InboundSetupField.from_dict(settings_item_data)



            settings.append(settings_item)


        steps = cast(list[str], d.pop("steps"))


        inbound_setup_view = cls(
            automatic=automatic,
            mx=mx,
            settings=settings,
            steps=steps,
        )

        return inbound_setup_view

