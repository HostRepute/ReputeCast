from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.provider_down_state import ProviderDownState
  from ..models.provider_incident_entry import ProviderIncidentEntry
  from ..models.provider_outage_stream import ProviderOutageStream
  from ..models.provider_quota_forecast import ProviderQuotaForecast
  from ..models.provider_why_not_move import ProviderWhyNotMove





T = TypeVar("T", bound="ProviderOutage")



@_attrs_define
class ProviderOutage:
    """ 
        Attributes:
            codes (list[str]):
            down (ProviderDownState):
            forecasts (list[ProviderQuotaForecast]):
            incidents (list[ProviderIncidentEntry]):
            operation (str):
            provider (str):
            provider_id (str):
            short (str):
            since (datetime.datetime):
            streams (list[ProviderOutageStream]):
            why_not (list[ProviderWhyNotMove]):
            error_rate (float | Unset): Percent of API calls that failed in the last 3 minutes; omitted when there were none
     """

    codes: list[str]
    down: ProviderDownState
    forecasts: list[ProviderQuotaForecast]
    incidents: list[ProviderIncidentEntry]
    operation: str
    provider: str
    provider_id: str
    short: str
    since: datetime.datetime
    streams: list[ProviderOutageStream]
    why_not: list[ProviderWhyNotMove]
    error_rate: float | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.provider_down_state import ProviderDownState # noqa: PLC0415
        from ..models.provider_incident_entry import ProviderIncidentEntry # noqa: PLC0415
        from ..models.provider_outage_stream import ProviderOutageStream # noqa: PLC0415
        from ..models.provider_quota_forecast import ProviderQuotaForecast # noqa: PLC0415
        from ..models.provider_why_not_move import ProviderWhyNotMove # noqa: PLC0415
        codes = self.codes



        down = self.down.to_dict()

        forecasts = []
        for forecasts_item_data in self.forecasts:
            forecasts_item = forecasts_item_data.to_dict()
            forecasts.append(forecasts_item)



        incidents = []
        for incidents_item_data in self.incidents:
            incidents_item = incidents_item_data.to_dict()
            incidents.append(incidents_item)



        operation = self.operation

        provider = self.provider

        provider_id = self.provider_id

        short = self.short

        since = self.since.isoformat()

        streams = []
        for streams_item_data in self.streams:
            streams_item = streams_item_data.to_dict()
            streams.append(streams_item)



        why_not = []
        for why_not_item_data in self.why_not:
            why_not_item = why_not_item_data.to_dict()
            why_not.append(why_not_item)



        error_rate = self.error_rate


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "codes": codes,
            "down": down,
            "forecasts": forecasts,
            "incidents": incidents,
            "operation": operation,
            "provider": provider,
            "providerId": provider_id,
            "short": short,
            "since": since,
            "streams": streams,
            "whyNot": why_not,
        })
        if error_rate is not UNSET:
            field_dict["errorRate"] = error_rate

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.provider_down_state import ProviderDownState # noqa: PLC0415
        from ..models.provider_incident_entry import ProviderIncidentEntry # noqa: PLC0415
        from ..models.provider_outage_stream import ProviderOutageStream # noqa: PLC0415
        from ..models.provider_quota_forecast import ProviderQuotaForecast # noqa: PLC0415
        from ..models.provider_why_not_move import ProviderWhyNotMove # noqa: PLC0415
        d = dict(src_dict)
        codes = cast(list[str], d.pop("codes"))


        down = ProviderDownState.from_dict(d.pop("down"))




        forecasts = []
        _forecasts = d.pop("forecasts")
        for forecasts_item_data in (_forecasts):
            forecasts_item = ProviderQuotaForecast.from_dict(forecasts_item_data)



            forecasts.append(forecasts_item)


        incidents = []
        _incidents = d.pop("incidents")
        for incidents_item_data in (_incidents):
            incidents_item = ProviderIncidentEntry.from_dict(incidents_item_data)



            incidents.append(incidents_item)


        operation = d.pop("operation")

        provider = d.pop("provider")

        provider_id = d.pop("providerId")

        short = d.pop("short")

        since = datetime.datetime.fromisoformat(d.pop("since"))




        streams = []
        _streams = d.pop("streams")
        for streams_item_data in (_streams):
            streams_item = ProviderOutageStream.from_dict(streams_item_data)



            streams.append(streams_item)


        why_not = []
        _why_not = d.pop("whyNot")
        for why_not_item_data in (_why_not):
            why_not_item = ProviderWhyNotMove.from_dict(why_not_item_data)



            why_not.append(why_not_item)


        error_rate = d.pop("errorRate", UNSET)

        provider_outage = cls(
            codes=codes,
            down=down,
            forecasts=forecasts,
            incidents=incidents,
            operation=operation,
            provider=provider,
            provider_id=provider_id,
            short=short,
            since=since,
            streams=streams,
            why_not=why_not,
            error_rate=error_rate,
        )

        return provider_outage

