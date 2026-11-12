from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.host_repute_connection import HostReputeConnection
  from ..models.host_repute_data_source import HostReputeDataSource
  from ..models.host_repute_integrations_mcp import HostReputeIntegrationsMCP
  from ..models.host_repute_integrations_webhooks import HostReputeIntegrationsWebhooks





T = TypeVar("T", bound="HostReputeIntegrationsOverview")



@_attrs_define
class HostReputeIntegrationsOverview:
    """ 
        Attributes:
            host_repute (HostReputeConnection):
            mcp (HostReputeIntegrationsMCP):
            postmaster (HostReputeDataSource):
            snds (HostReputeDataSource):
            webhooks (HostReputeIntegrationsWebhooks):
     """

    host_repute: HostReputeConnection
    mcp: HostReputeIntegrationsMCP
    postmaster: HostReputeDataSource
    snds: HostReputeDataSource
    webhooks: HostReputeIntegrationsWebhooks





    def to_dict(self) -> dict[str, Any]:
        from ..models.host_repute_connection import HostReputeConnection # noqa: PLC0415
        from ..models.host_repute_data_source import HostReputeDataSource # noqa: PLC0415
        from ..models.host_repute_integrations_mcp import HostReputeIntegrationsMCP # noqa: PLC0415
        from ..models.host_repute_integrations_webhooks import HostReputeIntegrationsWebhooks # noqa: PLC0415
        host_repute = self.host_repute.to_dict()

        mcp = self.mcp.to_dict()

        postmaster = self.postmaster.to_dict()

        snds = self.snds.to_dict()

        webhooks = self.webhooks.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "hostRepute": host_repute,
            "mcp": mcp,
            "postmaster": postmaster,
            "snds": snds,
            "webhooks": webhooks,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.host_repute_connection import HostReputeConnection # noqa: PLC0415
        from ..models.host_repute_data_source import HostReputeDataSource # noqa: PLC0415
        from ..models.host_repute_integrations_mcp import HostReputeIntegrationsMCP # noqa: PLC0415
        from ..models.host_repute_integrations_webhooks import HostReputeIntegrationsWebhooks # noqa: PLC0415
        d = dict(src_dict)
        host_repute = HostReputeConnection.from_dict(d.pop("hostRepute"))




        mcp = HostReputeIntegrationsMCP.from_dict(d.pop("mcp"))




        postmaster = HostReputeDataSource.from_dict(d.pop("postmaster"))




        snds = HostReputeDataSource.from_dict(d.pop("snds"))




        webhooks = HostReputeIntegrationsWebhooks.from_dict(d.pop("webhooks"))




        host_repute_integrations_overview = cls(
            host_repute=host_repute,
            mcp=mcp,
            postmaster=postmaster,
            snds=snds,
            webhooks=webhooks,
        )

        return host_repute_integrations_overview

