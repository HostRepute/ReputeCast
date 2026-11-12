from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.provider_card_account_kind import ProviderCardAccountKind
from ..models.provider_card_color import ProviderCardColor
from ..models.provider_card_ip_mode import ProviderCardIpMode
from ..models.provider_card_status import ProviderCardStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.provider_capability_badge import ProviderCapabilityBadge
  from ..models.provider_cost_to_date import ProviderCostToDate
  from ..models.provider_credential_field import ProviderCredentialField
  from ..models.provider_down_state import ProviderDownState
  from ..models.provider_ip_source import ProviderIPSource
  from ..models.provider_limit_rate import ProviderLimitRate
  from ..models.provider_quota_use import ProviderQuotaUse
  from ..models.provider_send_rate import ProviderSendRate
  from ..models.provider_stream_role import ProviderStreamRole
  from ..models.provider_tenant import ProviderTenant
  from ..models.provider_warmup_brief import ProviderWarmupBrief
  from ..models.provider_webhook_feed import ProviderWebhookFeed





T = TypeVar("T", bound="ProviderCard")



@_attrs_define
class ProviderCard:
    """ 
        Attributes:
            account (str):
            account_kind (ProviderCardAccountKind):
            adapter (str):
            bounce (ProviderLimitRate):
            capabilities (list[ProviderCapabilityBadge]):
            color (ProviderCardColor):
            complaint (ProviderLimitRate):
            connected_at (datetime.datetime):
            cost (ProviderCostToDate):
            credentials (list[ProviderCredentialField]):
            id (str):
            ip_mode (ProviderCardIpMode):
            ip_source (ProviderIPSource):
            ips (list[str]):
            label (str):
            name (str):
            quota (ProviderQuotaUse):
            rate (ProviderSendRate):
            roles (list[ProviderStreamRole]):
            short (str):
            status (ProviderCardStatus):
            taking_failover (list[str]):
            webhook (ProviderWebhookFeed):
            checked_at (datetime.datetime | Unset):
            down (ProviderDownState | Unset):
            error_rate_3_m (float | Unset): Percent of API calls that failed in the last 3 minutes; omitted when there were
                none
            error_rate_7_d (float | Unset): Percent of API calls that failed
            tenant (ProviderTenant | Unset):
            warmup (ProviderWarmupBrief | Unset):
     """

    account: str
    account_kind: ProviderCardAccountKind
    adapter: str
    bounce: ProviderLimitRate
    capabilities: list[ProviderCapabilityBadge]
    color: ProviderCardColor
    complaint: ProviderLimitRate
    connected_at: datetime.datetime
    cost: ProviderCostToDate
    credentials: list[ProviderCredentialField]
    id: str
    ip_mode: ProviderCardIpMode
    ip_source: ProviderIPSource
    ips: list[str]
    label: str
    name: str
    quota: ProviderQuotaUse
    rate: ProviderSendRate
    roles: list[ProviderStreamRole]
    short: str
    status: ProviderCardStatus
    taking_failover: list[str]
    webhook: ProviderWebhookFeed
    checked_at: datetime.datetime | Unset = UNSET
    down: ProviderDownState | Unset = UNSET
    error_rate_3_m: float | Unset = UNSET
    error_rate_7_d: float | Unset = UNSET
    tenant: ProviderTenant | Unset = UNSET
    warmup: ProviderWarmupBrief | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.provider_capability_badge import ProviderCapabilityBadge # noqa: PLC0415
        from ..models.provider_cost_to_date import ProviderCostToDate # noqa: PLC0415
        from ..models.provider_credential_field import ProviderCredentialField # noqa: PLC0415
        from ..models.provider_down_state import ProviderDownState # noqa: PLC0415
        from ..models.provider_ip_source import ProviderIPSource # noqa: PLC0415
        from ..models.provider_limit_rate import ProviderLimitRate # noqa: PLC0415
        from ..models.provider_quota_use import ProviderQuotaUse # noqa: PLC0415
        from ..models.provider_send_rate import ProviderSendRate # noqa: PLC0415
        from ..models.provider_stream_role import ProviderStreamRole # noqa: PLC0415
        from ..models.provider_tenant import ProviderTenant # noqa: PLC0415
        from ..models.provider_warmup_brief import ProviderWarmupBrief # noqa: PLC0415
        from ..models.provider_webhook_feed import ProviderWebhookFeed # noqa: PLC0415
        account = self.account

        account_kind = self.account_kind.value

        adapter = self.adapter

        bounce = self.bounce.to_dict()

        capabilities = []
        for capabilities_item_data in self.capabilities:
            capabilities_item = capabilities_item_data.to_dict()
            capabilities.append(capabilities_item)



        color = self.color.value

        complaint = self.complaint.to_dict()

        connected_at = self.connected_at.isoformat()

        cost = self.cost.to_dict()

        credentials = []
        for credentials_item_data in self.credentials:
            credentials_item = credentials_item_data.to_dict()
            credentials.append(credentials_item)



        id = self.id

        ip_mode = self.ip_mode.value

        ip_source = self.ip_source.to_dict()

        ips = self.ips



        label = self.label

        name = self.name

        quota = self.quota.to_dict()

        rate = self.rate.to_dict()

        roles = []
        for roles_item_data in self.roles:
            roles_item = roles_item_data.to_dict()
            roles.append(roles_item)



        short = self.short

        status = self.status.value

        taking_failover = self.taking_failover



        webhook = self.webhook.to_dict()

        checked_at: str | Unset = UNSET
        if not isinstance(self.checked_at, Unset):
            checked_at = self.checked_at.isoformat()

        down: dict[str, Any] | Unset = UNSET
        if not isinstance(self.down, Unset):
            down = self.down.to_dict()

        error_rate_3_m = self.error_rate_3_m

        error_rate_7_d = self.error_rate_7_d

        tenant: dict[str, Any] | Unset = UNSET
        if not isinstance(self.tenant, Unset):
            tenant = self.tenant.to_dict()

        warmup: dict[str, Any] | Unset = UNSET
        if not isinstance(self.warmup, Unset):
            warmup = self.warmup.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "account": account,
            "accountKind": account_kind,
            "adapter": adapter,
            "bounce": bounce,
            "capabilities": capabilities,
            "color": color,
            "complaint": complaint,
            "connectedAt": connected_at,
            "cost": cost,
            "credentials": credentials,
            "id": id,
            "ipMode": ip_mode,
            "ipSource": ip_source,
            "ips": ips,
            "label": label,
            "name": name,
            "quota": quota,
            "rate": rate,
            "roles": roles,
            "short": short,
            "status": status,
            "takingFailover": taking_failover,
            "webhook": webhook,
        })
        if checked_at is not UNSET:
            field_dict["checkedAt"] = checked_at
        if down is not UNSET:
            field_dict["down"] = down
        if error_rate_3_m is not UNSET:
            field_dict["errorRate3m"] = error_rate_3_m
        if error_rate_7_d is not UNSET:
            field_dict["errorRate7d"] = error_rate_7_d
        if tenant is not UNSET:
            field_dict["tenant"] = tenant
        if warmup is not UNSET:
            field_dict["warmup"] = warmup

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.provider_capability_badge import ProviderCapabilityBadge # noqa: PLC0415
        from ..models.provider_cost_to_date import ProviderCostToDate # noqa: PLC0415
        from ..models.provider_credential_field import ProviderCredentialField # noqa: PLC0415
        from ..models.provider_down_state import ProviderDownState # noqa: PLC0415
        from ..models.provider_ip_source import ProviderIPSource # noqa: PLC0415
        from ..models.provider_limit_rate import ProviderLimitRate # noqa: PLC0415
        from ..models.provider_quota_use import ProviderQuotaUse # noqa: PLC0415
        from ..models.provider_send_rate import ProviderSendRate # noqa: PLC0415
        from ..models.provider_stream_role import ProviderStreamRole # noqa: PLC0415
        from ..models.provider_tenant import ProviderTenant # noqa: PLC0415
        from ..models.provider_warmup_brief import ProviderWarmupBrief # noqa: PLC0415
        from ..models.provider_webhook_feed import ProviderWebhookFeed # noqa: PLC0415
        d = dict(src_dict)
        account = d.pop("account")

        account_kind = ProviderCardAccountKind(d.pop("accountKind"))




        adapter = d.pop("adapter")

        bounce = ProviderLimitRate.from_dict(d.pop("bounce"))




        capabilities = []
        _capabilities = d.pop("capabilities")
        for capabilities_item_data in (_capabilities):
            capabilities_item = ProviderCapabilityBadge.from_dict(capabilities_item_data)



            capabilities.append(capabilities_item)


        color = ProviderCardColor(d.pop("color"))




        complaint = ProviderLimitRate.from_dict(d.pop("complaint"))




        connected_at = datetime.datetime.fromisoformat(d.pop("connectedAt"))




        cost = ProviderCostToDate.from_dict(d.pop("cost"))




        credentials = []
        _credentials = d.pop("credentials")
        for credentials_item_data in (_credentials):
            credentials_item = ProviderCredentialField.from_dict(credentials_item_data)



            credentials.append(credentials_item)


        id = d.pop("id")

        ip_mode = ProviderCardIpMode(d.pop("ipMode"))




        ip_source = ProviderIPSource.from_dict(d.pop("ipSource"))




        ips = cast(list[str], d.pop("ips"))


        label = d.pop("label")

        name = d.pop("name")

        quota = ProviderQuotaUse.from_dict(d.pop("quota"))




        rate = ProviderSendRate.from_dict(d.pop("rate"))




        roles = []
        _roles = d.pop("roles")
        for roles_item_data in (_roles):
            roles_item = ProviderStreamRole.from_dict(roles_item_data)



            roles.append(roles_item)


        short = d.pop("short")

        status = ProviderCardStatus(d.pop("status"))




        taking_failover = cast(list[str], d.pop("takingFailover"))


        webhook = ProviderWebhookFeed.from_dict(d.pop("webhook"))




        _checked_at = d.pop("checkedAt", UNSET)
        checked_at: datetime.datetime | Unset
        if isinstance(_checked_at,  Unset):
            checked_at = UNSET
        else:
            checked_at = datetime.datetime.fromisoformat(_checked_at)




        _down = d.pop("down", UNSET)
        down: ProviderDownState | Unset
        if isinstance(_down,  Unset):
            down = UNSET
        else:
            down = ProviderDownState.from_dict(_down)




        error_rate_3_m = d.pop("errorRate3m", UNSET)

        error_rate_7_d = d.pop("errorRate7d", UNSET)

        _tenant = d.pop("tenant", UNSET)
        tenant: ProviderTenant | Unset
        if isinstance(_tenant,  Unset):
            tenant = UNSET
        else:
            tenant = ProviderTenant.from_dict(_tenant)




        _warmup = d.pop("warmup", UNSET)
        warmup: ProviderWarmupBrief | Unset
        if isinstance(_warmup,  Unset):
            warmup = UNSET
        else:
            warmup = ProviderWarmupBrief.from_dict(_warmup)




        provider_card = cls(
            account=account,
            account_kind=account_kind,
            adapter=adapter,
            bounce=bounce,
            capabilities=capabilities,
            color=color,
            complaint=complaint,
            connected_at=connected_at,
            cost=cost,
            credentials=credentials,
            id=id,
            ip_mode=ip_mode,
            ip_source=ip_source,
            ips=ips,
            label=label,
            name=name,
            quota=quota,
            rate=rate,
            roles=roles,
            short=short,
            status=status,
            taking_failover=taking_failover,
            webhook=webhook,
            checked_at=checked_at,
            down=down,
            error_rate_3_m=error_rate_3_m,
            error_rate_7_d=error_rate_7_d,
            tenant=tenant,
            warmup=warmup,
        )

        return provider_card

