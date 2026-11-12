from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.verification_service_view_key import VerificationServiceViewKey
from ..models.verification_service_view_status import VerificationServiceViewStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.verification_capabilities import VerificationCapabilities
  from ..models.verification_mapping import VerificationMapping
  from ..models.verification_region import VerificationRegion





T = TypeVar("T", bound="VerificationServiceView")



@_attrs_define
class VerificationServiceView:
    """ 
        Attributes:
            bulk_uses_30_d (int):
            capabilities (VerificationCapabilities):
            checks30d (int):
            connected (bool):
            default_bulk (bool):
            default_forms (bool):
            down (bool):
            initials (str):
            key (VerificationServiceViewKey):
            key_help (str):
            mappings (list[VerificationMapping]):
            name (str):
            price (float): Price per check in dollars, from the workspace's plan
            region (str):
            region_label (str):
            regions (list[VerificationRegion]):
            spent30d (float):
            status (VerificationServiceViewStatus):
            website (str):
            balance (int | Unset):
            balance_at (datetime.datetime | Unset):
            connected_at (datetime.datetime | Unset):
            connected_by (str | Unset):
            last_error (str | Unset):
            last_error_at (datetime.datetime | Unset):
            last_error_since (datetime.datetime | Unset):
            last_success_at (datetime.datetime | Unset):
            last_success_what (str | Unset):
            last_used_at (datetime.datetime | Unset):
            last_used_by (str | Unset):
            last_used_what (str | Unset):
            needed (int | Unset):
            needed_label (str | Unset):
            paused_job (str | Unset):
            price_by (str | Unset): Who entered the price
            test_ok (bool | Unset):
            tested_at (datetime.datetime | Unset):
     """

    bulk_uses_30_d: int
    capabilities: VerificationCapabilities
    checks30d: int
    connected: bool
    default_bulk: bool
    default_forms: bool
    down: bool
    initials: str
    key: VerificationServiceViewKey
    key_help: str
    mappings: list[VerificationMapping]
    name: str
    price: float
    region: str
    region_label: str
    regions: list[VerificationRegion]
    spent30d: float
    status: VerificationServiceViewStatus
    website: str
    balance: int | Unset = UNSET
    balance_at: datetime.datetime | Unset = UNSET
    connected_at: datetime.datetime | Unset = UNSET
    connected_by: str | Unset = UNSET
    last_error: str | Unset = UNSET
    last_error_at: datetime.datetime | Unset = UNSET
    last_error_since: datetime.datetime | Unset = UNSET
    last_success_at: datetime.datetime | Unset = UNSET
    last_success_what: str | Unset = UNSET
    last_used_at: datetime.datetime | Unset = UNSET
    last_used_by: str | Unset = UNSET
    last_used_what: str | Unset = UNSET
    needed: int | Unset = UNSET
    needed_label: str | Unset = UNSET
    paused_job: str | Unset = UNSET
    price_by: str | Unset = UNSET
    test_ok: bool | Unset = UNSET
    tested_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.verification_capabilities import VerificationCapabilities # noqa: PLC0415
        from ..models.verification_mapping import VerificationMapping # noqa: PLC0415
        from ..models.verification_region import VerificationRegion # noqa: PLC0415
        bulk_uses_30_d = self.bulk_uses_30_d

        capabilities = self.capabilities.to_dict()

        checks30d = self.checks30d

        connected = self.connected

        default_bulk = self.default_bulk

        default_forms = self.default_forms

        down = self.down

        initials = self.initials

        key = self.key.value

        key_help = self.key_help

        mappings = []
        for mappings_item_data in self.mappings:
            mappings_item = mappings_item_data.to_dict()
            mappings.append(mappings_item)



        name = self.name

        price = self.price

        region = self.region

        region_label = self.region_label

        regions = []
        for regions_item_data in self.regions:
            regions_item = regions_item_data.to_dict()
            regions.append(regions_item)



        spent30d = self.spent30d

        status = self.status.value

        website = self.website

        balance = self.balance

        balance_at: str | Unset = UNSET
        if not isinstance(self.balance_at, Unset):
            balance_at = self.balance_at.isoformat()

        connected_at: str | Unset = UNSET
        if not isinstance(self.connected_at, Unset):
            connected_at = self.connected_at.isoformat()

        connected_by = self.connected_by

        last_error = self.last_error

        last_error_at: str | Unset = UNSET
        if not isinstance(self.last_error_at, Unset):
            last_error_at = self.last_error_at.isoformat()

        last_error_since: str | Unset = UNSET
        if not isinstance(self.last_error_since, Unset):
            last_error_since = self.last_error_since.isoformat()

        last_success_at: str | Unset = UNSET
        if not isinstance(self.last_success_at, Unset):
            last_success_at = self.last_success_at.isoformat()

        last_success_what = self.last_success_what

        last_used_at: str | Unset = UNSET
        if not isinstance(self.last_used_at, Unset):
            last_used_at = self.last_used_at.isoformat()

        last_used_by = self.last_used_by

        last_used_what = self.last_used_what

        needed = self.needed

        needed_label = self.needed_label

        paused_job = self.paused_job

        price_by = self.price_by

        test_ok = self.test_ok

        tested_at: str | Unset = UNSET
        if not isinstance(self.tested_at, Unset):
            tested_at = self.tested_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "bulkUses30d": bulk_uses_30_d,
            "capabilities": capabilities,
            "checks30d": checks30d,
            "connected": connected,
            "defaultBulk": default_bulk,
            "defaultForms": default_forms,
            "down": down,
            "initials": initials,
            "key": key,
            "keyHelp": key_help,
            "mappings": mappings,
            "name": name,
            "price": price,
            "region": region,
            "regionLabel": region_label,
            "regions": regions,
            "spent30d": spent30d,
            "status": status,
            "website": website,
        })
        if balance is not UNSET:
            field_dict["balance"] = balance
        if balance_at is not UNSET:
            field_dict["balanceAt"] = balance_at
        if connected_at is not UNSET:
            field_dict["connectedAt"] = connected_at
        if connected_by is not UNSET:
            field_dict["connectedBy"] = connected_by
        if last_error is not UNSET:
            field_dict["lastError"] = last_error
        if last_error_at is not UNSET:
            field_dict["lastErrorAt"] = last_error_at
        if last_error_since is not UNSET:
            field_dict["lastErrorSince"] = last_error_since
        if last_success_at is not UNSET:
            field_dict["lastSuccessAt"] = last_success_at
        if last_success_what is not UNSET:
            field_dict["lastSuccessWhat"] = last_success_what
        if last_used_at is not UNSET:
            field_dict["lastUsedAt"] = last_used_at
        if last_used_by is not UNSET:
            field_dict["lastUsedBy"] = last_used_by
        if last_used_what is not UNSET:
            field_dict["lastUsedWhat"] = last_used_what
        if needed is not UNSET:
            field_dict["needed"] = needed
        if needed_label is not UNSET:
            field_dict["neededLabel"] = needed_label
        if paused_job is not UNSET:
            field_dict["pausedJob"] = paused_job
        if price_by is not UNSET:
            field_dict["priceBy"] = price_by
        if test_ok is not UNSET:
            field_dict["testOk"] = test_ok
        if tested_at is not UNSET:
            field_dict["testedAt"] = tested_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.verification_capabilities import VerificationCapabilities # noqa: PLC0415
        from ..models.verification_mapping import VerificationMapping # noqa: PLC0415
        from ..models.verification_region import VerificationRegion # noqa: PLC0415
        d = dict(src_dict)
        bulk_uses_30_d = d.pop("bulkUses30d")

        capabilities = VerificationCapabilities.from_dict(d.pop("capabilities"))




        checks30d = d.pop("checks30d")

        connected = d.pop("connected")

        default_bulk = d.pop("defaultBulk")

        default_forms = d.pop("defaultForms")

        down = d.pop("down")

        initials = d.pop("initials")

        key = VerificationServiceViewKey(d.pop("key"))




        key_help = d.pop("keyHelp")

        mappings = []
        _mappings = d.pop("mappings")
        for mappings_item_data in (_mappings):
            mappings_item = VerificationMapping.from_dict(mappings_item_data)



            mappings.append(mappings_item)


        name = d.pop("name")

        price = d.pop("price")

        region = d.pop("region")

        region_label = d.pop("regionLabel")

        regions = []
        _regions = d.pop("regions")
        for regions_item_data in (_regions):
            regions_item = VerificationRegion.from_dict(regions_item_data)



            regions.append(regions_item)


        spent30d = d.pop("spent30d")

        status = VerificationServiceViewStatus(d.pop("status"))




        website = d.pop("website")

        balance = d.pop("balance", UNSET)

        _balance_at = d.pop("balanceAt", UNSET)
        balance_at: datetime.datetime | Unset
        if isinstance(_balance_at,  Unset):
            balance_at = UNSET
        else:
            balance_at = datetime.datetime.fromisoformat(_balance_at)




        _connected_at = d.pop("connectedAt", UNSET)
        connected_at: datetime.datetime | Unset
        if isinstance(_connected_at,  Unset):
            connected_at = UNSET
        else:
            connected_at = datetime.datetime.fromisoformat(_connected_at)




        connected_by = d.pop("connectedBy", UNSET)

        last_error = d.pop("lastError", UNSET)

        _last_error_at = d.pop("lastErrorAt", UNSET)
        last_error_at: datetime.datetime | Unset
        if isinstance(_last_error_at,  Unset):
            last_error_at = UNSET
        else:
            last_error_at = datetime.datetime.fromisoformat(_last_error_at)




        _last_error_since = d.pop("lastErrorSince", UNSET)
        last_error_since: datetime.datetime | Unset
        if isinstance(_last_error_since,  Unset):
            last_error_since = UNSET
        else:
            last_error_since = datetime.datetime.fromisoformat(_last_error_since)




        _last_success_at = d.pop("lastSuccessAt", UNSET)
        last_success_at: datetime.datetime | Unset
        if isinstance(_last_success_at,  Unset):
            last_success_at = UNSET
        else:
            last_success_at = datetime.datetime.fromisoformat(_last_success_at)




        last_success_what = d.pop("lastSuccessWhat", UNSET)

        _last_used_at = d.pop("lastUsedAt", UNSET)
        last_used_at: datetime.datetime | Unset
        if isinstance(_last_used_at,  Unset):
            last_used_at = UNSET
        else:
            last_used_at = datetime.datetime.fromisoformat(_last_used_at)




        last_used_by = d.pop("lastUsedBy", UNSET)

        last_used_what = d.pop("lastUsedWhat", UNSET)

        needed = d.pop("needed", UNSET)

        needed_label = d.pop("neededLabel", UNSET)

        paused_job = d.pop("pausedJob", UNSET)

        price_by = d.pop("priceBy", UNSET)

        test_ok = d.pop("testOk", UNSET)

        _tested_at = d.pop("testedAt", UNSET)
        tested_at: datetime.datetime | Unset
        if isinstance(_tested_at,  Unset):
            tested_at = UNSET
        else:
            tested_at = datetime.datetime.fromisoformat(_tested_at)




        verification_service_view = cls(
            bulk_uses_30_d=bulk_uses_30_d,
            capabilities=capabilities,
            checks30d=checks30d,
            connected=connected,
            default_bulk=default_bulk,
            default_forms=default_forms,
            down=down,
            initials=initials,
            key=key,
            key_help=key_help,
            mappings=mappings,
            name=name,
            price=price,
            region=region,
            region_label=region_label,
            regions=regions,
            spent30d=spent30d,
            status=status,
            website=website,
            balance=balance,
            balance_at=balance_at,
            connected_at=connected_at,
            connected_by=connected_by,
            last_error=last_error,
            last_error_at=last_error_at,
            last_error_since=last_error_since,
            last_success_at=last_success_at,
            last_success_what=last_success_what,
            last_used_at=last_used_at,
            last_used_by=last_used_by,
            last_used_what=last_used_what,
            needed=needed,
            needed_label=needed_label,
            paused_job=paused_job,
            price_by=price_by,
            test_ok=test_ok,
            tested_at=tested_at,
        )

        return verification_service_view

