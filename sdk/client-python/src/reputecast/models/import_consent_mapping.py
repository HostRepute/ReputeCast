from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.import_consent_mapping_evidence import ImportConsentMappingEvidence
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.import_consent_status import ImportConsentStatus
  from ..models.import_sms_consent import ImportSmsConsent





T = TypeVar("T", bound="ImportConsentMapping")



@_attrs_define
class ImportConsentMapping:
    """ 
        Attributes:
            double_opt_in (bool):
            evidence (ImportConsentMappingEvidence):
            evidence_source (str):
            history_example (str):
            in_sync (int):
            providers (list[str]):
            pull_suppressions (bool):
            record_history (bool):
            statuses (list[ImportConsentStatus]):
            suppress_count (int):
            tag (str):
            sms (ImportSmsConsent | Unset):
     """

    double_opt_in: bool
    evidence: ImportConsentMappingEvidence
    evidence_source: str
    history_example: str
    in_sync: int
    providers: list[str]
    pull_suppressions: bool
    record_history: bool
    statuses: list[ImportConsentStatus]
    suppress_count: int
    tag: str
    sms: ImportSmsConsent | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.import_consent_status import ImportConsentStatus # noqa: PLC0415
        from ..models.import_sms_consent import ImportSmsConsent # noqa: PLC0415
        double_opt_in = self.double_opt_in

        evidence = self.evidence.value

        evidence_source = self.evidence_source

        history_example = self.history_example

        in_sync = self.in_sync

        providers = self.providers



        pull_suppressions = self.pull_suppressions

        record_history = self.record_history

        statuses = []
        for statuses_item_data in self.statuses:
            statuses_item = statuses_item_data.to_dict()
            statuses.append(statuses_item)



        suppress_count = self.suppress_count

        tag = self.tag

        sms: dict[str, Any] | Unset = UNSET
        if not isinstance(self.sms, Unset):
            sms = self.sms.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "doubleOptIn": double_opt_in,
            "evidence": evidence,
            "evidenceSource": evidence_source,
            "historyExample": history_example,
            "inSync": in_sync,
            "providers": providers,
            "pullSuppressions": pull_suppressions,
            "recordHistory": record_history,
            "statuses": statuses,
            "suppressCount": suppress_count,
            "tag": tag,
        })
        if sms is not UNSET:
            field_dict["sms"] = sms

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.import_consent_status import ImportConsentStatus # noqa: PLC0415
        from ..models.import_sms_consent import ImportSmsConsent # noqa: PLC0415
        d = dict(src_dict)
        double_opt_in = d.pop("doubleOptIn")

        evidence = ImportConsentMappingEvidence(d.pop("evidence"))




        evidence_source = d.pop("evidenceSource")

        history_example = d.pop("historyExample")

        in_sync = d.pop("inSync")

        providers = cast(list[str], d.pop("providers"))


        pull_suppressions = d.pop("pullSuppressions")

        record_history = d.pop("recordHistory")

        statuses = []
        _statuses = d.pop("statuses")
        for statuses_item_data in (_statuses):
            statuses_item = ImportConsentStatus.from_dict(statuses_item_data)



            statuses.append(statuses_item)


        suppress_count = d.pop("suppressCount")

        tag = d.pop("tag")

        _sms = d.pop("sms", UNSET)
        sms: ImportSmsConsent | Unset
        if isinstance(_sms,  Unset):
            sms = UNSET
        else:
            sms = ImportSmsConsent.from_dict(_sms)




        import_consent_mapping = cls(
            double_opt_in=double_opt_in,
            evidence=evidence,
            evidence_source=evidence_source,
            history_example=history_example,
            in_sync=in_sync,
            providers=providers,
            pull_suppressions=pull_suppressions,
            record_history=record_history,
            statuses=statuses,
            suppress_count=suppress_count,
            tag=tag,
            sms=sms,
        )

        return import_consent_mapping

