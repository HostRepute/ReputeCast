from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.survey_link_via import SurveyLinkVia
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="SurveyLink")



@_attrs_define
class SurveyLink:
    """ 
        Attributes:
            campaign_id (str):
            name (str):
            status (str):
            via (SurveyLinkVia):
            recipients (int | Unset):
            sent_at (datetime.datetime | Unset):
     """

    campaign_id: str
    name: str
    status: str
    via: SurveyLinkVia
    recipients: int | Unset = UNSET
    sent_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        campaign_id = self.campaign_id

        name = self.name

        status = self.status

        via = self.via.value

        recipients = self.recipients

        sent_at: str | Unset = UNSET
        if not isinstance(self.sent_at, Unset):
            sent_at = self.sent_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "campaignId": campaign_id,
            "name": name,
            "status": status,
            "via": via,
        })
        if recipients is not UNSET:
            field_dict["recipients"] = recipients
        if sent_at is not UNSET:
            field_dict["sentAt"] = sent_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        campaign_id = d.pop("campaignId")

        name = d.pop("name")

        status = d.pop("status")

        via = SurveyLinkVia(d.pop("via"))




        recipients = d.pop("recipients", UNSET)

        _sent_at = d.pop("sentAt", UNSET)
        sent_at: datetime.datetime | Unset
        if isinstance(_sent_at,  Unset):
            sent_at = UNSET
        else:
            sent_at = datetime.datetime.fromisoformat(_sent_at)




        survey_link = cls(
            campaign_id=campaign_id,
            name=name,
            status=status,
            via=via,
            recipients=recipients,
            sent_at=sent_at,
        )

        return survey_link

