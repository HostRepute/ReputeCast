from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.whats_app_campaign_over_limit import WhatsAppCampaignOverLimit
from ..models.whats_app_campaign_status import WhatsAppCampaignStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.whats_app_access import WhatsAppAccess
  from ..models.whats_app_audience import WhatsAppAudience
  from ..models.whats_app_business import WhatsAppBusiness
  from ..models.whats_app_cost import WhatsAppCost
  from ..models.whats_app_field import WhatsAppField
  from ..models.whats_app_list_consent import WhatsAppListConsent
  from ..models.whats_app_preview import WhatsAppPreview
  from ..models.whats_app_schedule import WhatsAppSchedule
  from ..models.whats_app_template import WhatsAppTemplate
  from ..models.whats_app_test_result import WhatsAppTestResult
  from ..models.whats_app_variable import WhatsAppVariable





T = TypeVar("T", bound="WhatsAppCampaign")



@_attrs_define
class WhatsAppCampaign:
    """ 
        Attributes:
            access (WhatsAppAccess):
            account_status (str):
            audience (WhatsAppAudience):
            connected (bool):
            cost (WhatsAppCost):
            created_by (str):
            fields (list[WhatsAppField]):
            id (str):
            list_options (list[WhatsAppListConsent]):
            name (str):
            over_limit (WhatsAppCampaignOverLimit):
            quiet (bool):
            schedule (WhatsAppSchedule):
            segments (list[str]):
            status (WhatsAppCampaignStatus):
            templates (list[WhatsAppTemplate]):
            updated_at (datetime.datetime):
            updated_by (str):
            variables (list[WhatsAppVariable]):
            version (int):
            window (str):
            autosaved_at (datetime.datetime | Unset):
            business (WhatsAppBusiness | Unset):
            preview (WhatsAppPreview | Unset):
            template (WhatsAppTemplate | Unset):
            template_id (str | Unset):
            test (WhatsAppTestResult | Unset):
     """

    access: WhatsAppAccess
    account_status: str
    audience: WhatsAppAudience
    connected: bool
    cost: WhatsAppCost
    created_by: str
    fields: list[WhatsAppField]
    id: str
    list_options: list[WhatsAppListConsent]
    name: str
    over_limit: WhatsAppCampaignOverLimit
    quiet: bool
    schedule: WhatsAppSchedule
    segments: list[str]
    status: WhatsAppCampaignStatus
    templates: list[WhatsAppTemplate]
    updated_at: datetime.datetime
    updated_by: str
    variables: list[WhatsAppVariable]
    version: int
    window: str
    autosaved_at: datetime.datetime | Unset = UNSET
    business: WhatsAppBusiness | Unset = UNSET
    preview: WhatsAppPreview | Unset = UNSET
    template: WhatsAppTemplate | Unset = UNSET
    template_id: str | Unset = UNSET
    test: WhatsAppTestResult | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.whats_app_access import WhatsAppAccess # noqa: PLC0415
        from ..models.whats_app_audience import WhatsAppAudience # noqa: PLC0415
        from ..models.whats_app_business import WhatsAppBusiness # noqa: PLC0415
        from ..models.whats_app_cost import WhatsAppCost # noqa: PLC0415
        from ..models.whats_app_field import WhatsAppField # noqa: PLC0415
        from ..models.whats_app_list_consent import WhatsAppListConsent # noqa: PLC0415
        from ..models.whats_app_preview import WhatsAppPreview # noqa: PLC0415
        from ..models.whats_app_schedule import WhatsAppSchedule # noqa: PLC0415
        from ..models.whats_app_template import WhatsAppTemplate # noqa: PLC0415
        from ..models.whats_app_test_result import WhatsAppTestResult # noqa: PLC0415
        from ..models.whats_app_variable import WhatsAppVariable # noqa: PLC0415
        access = self.access.to_dict()

        account_status = self.account_status

        audience = self.audience.to_dict()

        connected = self.connected

        cost = self.cost.to_dict()

        created_by = self.created_by

        fields = []
        for fields_item_data in self.fields:
            fields_item = fields_item_data.to_dict()
            fields.append(fields_item)



        id = self.id

        list_options = []
        for list_options_item_data in self.list_options:
            list_options_item = list_options_item_data.to_dict()
            list_options.append(list_options_item)



        name = self.name

        over_limit = self.over_limit.value

        quiet = self.quiet

        schedule = self.schedule.to_dict()

        segments = self.segments



        status = self.status.value

        templates = []
        for templates_item_data in self.templates:
            templates_item = templates_item_data.to_dict()
            templates.append(templates_item)



        updated_at = self.updated_at.isoformat()

        updated_by = self.updated_by

        variables = []
        for variables_item_data in self.variables:
            variables_item = variables_item_data.to_dict()
            variables.append(variables_item)



        version = self.version

        window = self.window

        autosaved_at: str | Unset = UNSET
        if not isinstance(self.autosaved_at, Unset):
            autosaved_at = self.autosaved_at.isoformat()

        business: dict[str, Any] | Unset = UNSET
        if not isinstance(self.business, Unset):
            business = self.business.to_dict()

        preview: dict[str, Any] | Unset = UNSET
        if not isinstance(self.preview, Unset):
            preview = self.preview.to_dict()

        template: dict[str, Any] | Unset = UNSET
        if not isinstance(self.template, Unset):
            template = self.template.to_dict()

        template_id = self.template_id

        test: dict[str, Any] | Unset = UNSET
        if not isinstance(self.test, Unset):
            test = self.test.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "access": access,
            "accountStatus": account_status,
            "audience": audience,
            "connected": connected,
            "cost": cost,
            "createdBy": created_by,
            "fields": fields,
            "id": id,
            "listOptions": list_options,
            "name": name,
            "overLimit": over_limit,
            "quiet": quiet,
            "schedule": schedule,
            "segments": segments,
            "status": status,
            "templates": templates,
            "updatedAt": updated_at,
            "updatedBy": updated_by,
            "variables": variables,
            "version": version,
            "window": window,
        })
        if autosaved_at is not UNSET:
            field_dict["autosavedAt"] = autosaved_at
        if business is not UNSET:
            field_dict["business"] = business
        if preview is not UNSET:
            field_dict["preview"] = preview
        if template is not UNSET:
            field_dict["template"] = template
        if template_id is not UNSET:
            field_dict["templateId"] = template_id
        if test is not UNSET:
            field_dict["test"] = test

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.whats_app_access import WhatsAppAccess # noqa: PLC0415
        from ..models.whats_app_audience import WhatsAppAudience # noqa: PLC0415
        from ..models.whats_app_business import WhatsAppBusiness # noqa: PLC0415
        from ..models.whats_app_cost import WhatsAppCost # noqa: PLC0415
        from ..models.whats_app_field import WhatsAppField # noqa: PLC0415
        from ..models.whats_app_list_consent import WhatsAppListConsent # noqa: PLC0415
        from ..models.whats_app_preview import WhatsAppPreview # noqa: PLC0415
        from ..models.whats_app_schedule import WhatsAppSchedule # noqa: PLC0415
        from ..models.whats_app_template import WhatsAppTemplate # noqa: PLC0415
        from ..models.whats_app_test_result import WhatsAppTestResult # noqa: PLC0415
        from ..models.whats_app_variable import WhatsAppVariable # noqa: PLC0415
        d = dict(src_dict)
        access = WhatsAppAccess.from_dict(d.pop("access"))




        account_status = d.pop("accountStatus")

        audience = WhatsAppAudience.from_dict(d.pop("audience"))




        connected = d.pop("connected")

        cost = WhatsAppCost.from_dict(d.pop("cost"))




        created_by = d.pop("createdBy")

        fields = []
        _fields = d.pop("fields")
        for fields_item_data in (_fields):
            fields_item = WhatsAppField.from_dict(fields_item_data)



            fields.append(fields_item)


        id = d.pop("id")

        list_options = []
        _list_options = d.pop("listOptions")
        for list_options_item_data in (_list_options):
            list_options_item = WhatsAppListConsent.from_dict(list_options_item_data)



            list_options.append(list_options_item)


        name = d.pop("name")

        over_limit = WhatsAppCampaignOverLimit(d.pop("overLimit"))




        quiet = d.pop("quiet")

        schedule = WhatsAppSchedule.from_dict(d.pop("schedule"))




        segments = cast(list[str], d.pop("segments"))


        status = WhatsAppCampaignStatus(d.pop("status"))




        templates = []
        _templates = d.pop("templates")
        for templates_item_data in (_templates):
            templates_item = WhatsAppTemplate.from_dict(templates_item_data)



            templates.append(templates_item)


        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))




        updated_by = d.pop("updatedBy")

        variables = []
        _variables = d.pop("variables")
        for variables_item_data in (_variables):
            variables_item = WhatsAppVariable.from_dict(variables_item_data)



            variables.append(variables_item)


        version = d.pop("version")

        window = d.pop("window")

        _autosaved_at = d.pop("autosavedAt", UNSET)
        autosaved_at: datetime.datetime | Unset
        if isinstance(_autosaved_at,  Unset):
            autosaved_at = UNSET
        else:
            autosaved_at = datetime.datetime.fromisoformat(_autosaved_at)




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




        _template = d.pop("template", UNSET)
        template: WhatsAppTemplate | Unset
        if isinstance(_template,  Unset):
            template = UNSET
        else:
            template = WhatsAppTemplate.from_dict(_template)




        template_id = d.pop("templateId", UNSET)

        _test = d.pop("test", UNSET)
        test: WhatsAppTestResult | Unset
        if isinstance(_test,  Unset):
            test = UNSET
        else:
            test = WhatsAppTestResult.from_dict(_test)




        whats_app_campaign = cls(
            access=access,
            account_status=account_status,
            audience=audience,
            connected=connected,
            cost=cost,
            created_by=created_by,
            fields=fields,
            id=id,
            list_options=list_options,
            name=name,
            over_limit=over_limit,
            quiet=quiet,
            schedule=schedule,
            segments=segments,
            status=status,
            templates=templates,
            updated_at=updated_at,
            updated_by=updated_by,
            variables=variables,
            version=version,
            window=window,
            autosaved_at=autosaved_at,
            business=business,
            preview=preview,
            template=template,
            template_id=template_id,
            test=test,
        )

        return whats_app_campaign

