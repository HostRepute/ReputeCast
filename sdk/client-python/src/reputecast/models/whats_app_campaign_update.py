from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.whats_app_campaign_update_mode import WhatsAppCampaignUpdateMode
from ..models.whats_app_campaign_update_over_limit import WhatsAppCampaignUpdateOverLimit
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.whats_app_variable import WhatsAppVariable





T = TypeVar("T", bound="WhatsAppCampaignUpdate")



@_attrs_define
class WhatsAppCampaignUpdate:
    """ 
        Attributes:
            lists (list[str]):
            mode (WhatsAppCampaignUpdateMode):
            name (str):
            template_id (str):
            variables (list[WhatsAppVariable]):
            version (int):
            local_date (str | Unset):
            local_minute (int | Unset):
            over_limit (WhatsAppCampaignUpdateOverLimit | Unset):
            segments (list[str] | Unset):
            send_at (datetime.datetime | Unset):
     """

    lists: list[str]
    mode: WhatsAppCampaignUpdateMode
    name: str
    template_id: str
    variables: list[WhatsAppVariable]
    version: int
    local_date: str | Unset = UNSET
    local_minute: int | Unset = UNSET
    over_limit: WhatsAppCampaignUpdateOverLimit | Unset = UNSET
    segments: list[str] | Unset = UNSET
    send_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.whats_app_variable import WhatsAppVariable # noqa: PLC0415
        lists = self.lists



        mode = self.mode.value

        name = self.name

        template_id = self.template_id

        variables = []
        for variables_item_data in self.variables:
            variables_item = variables_item_data.to_dict()
            variables.append(variables_item)



        version = self.version

        local_date = self.local_date

        local_minute = self.local_minute

        over_limit: str | Unset = UNSET
        if not isinstance(self.over_limit, Unset):
            over_limit = self.over_limit.value


        segments: list[str] | Unset = UNSET
        if not isinstance(self.segments, Unset):
            segments = self.segments



        send_at: str | Unset = UNSET
        if not isinstance(self.send_at, Unset):
            send_at = self.send_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "lists": lists,
            "mode": mode,
            "name": name,
            "templateId": template_id,
            "variables": variables,
            "version": version,
        })
        if local_date is not UNSET:
            field_dict["localDate"] = local_date
        if local_minute is not UNSET:
            field_dict["localMinute"] = local_minute
        if over_limit is not UNSET:
            field_dict["overLimit"] = over_limit
        if segments is not UNSET:
            field_dict["segments"] = segments
        if send_at is not UNSET:
            field_dict["sendAt"] = send_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.whats_app_variable import WhatsAppVariable # noqa: PLC0415
        d = dict(src_dict)
        lists = cast(list[str], d.pop("lists"))


        mode = WhatsAppCampaignUpdateMode(d.pop("mode"))




        name = d.pop("name")

        template_id = d.pop("templateId")

        variables = []
        _variables = d.pop("variables")
        for variables_item_data in (_variables):
            variables_item = WhatsAppVariable.from_dict(variables_item_data)



            variables.append(variables_item)


        version = d.pop("version")

        local_date = d.pop("localDate", UNSET)

        local_minute = d.pop("localMinute", UNSET)

        _over_limit = d.pop("overLimit", UNSET)
        over_limit: WhatsAppCampaignUpdateOverLimit | Unset
        if isinstance(_over_limit,  Unset):
            over_limit = UNSET
        else:
            over_limit = WhatsAppCampaignUpdateOverLimit(_over_limit)




        segments = cast(list[str], d.pop("segments", UNSET))


        _send_at = d.pop("sendAt", UNSET)
        send_at: datetime.datetime | Unset
        if isinstance(_send_at,  Unset):
            send_at = UNSET
        else:
            send_at = datetime.datetime.fromisoformat(_send_at)




        whats_app_campaign_update = cls(
            lists=lists,
            mode=mode,
            name=name,
            template_id=template_id,
            variables=variables,
            version=version,
            local_date=local_date,
            local_minute=local_minute,
            over_limit=over_limit,
            segments=segments,
            send_at=send_at,
        )

        return whats_app_campaign_update

