import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { companyFormSchema, saveCompany } from './api'
import type { Company, CompanyBody } from './api'
import { Dialog } from '../../components/admin/Dialog'
import { FormFooter, FieldError } from '../../components/admin/FormFooter'
import { useAdminMutation } from '../../components/admin/useAdminMutation'
export function CompanyForm({
  company,
  onClose,
}: {
  company?: Company
  onClose: () => void
}) {
  const form = useForm<CompanyBody>({
    resolver: zodResolver(companyFormSchema),
    defaultValues: {
      name: company?.name ?? '',
      reference: company?.reference ?? '',
      active: company?.active ?? true,
    },
  })
  const mutation = useAdminMutation<CompanyBody>(
    (body) => saveCompany(company?.id, body),
    onClose,
  )
  return (
    <Dialog
      title={company ? 'Editar empresa' : 'Nova empresa'}
      onClose={onClose}
      busy={mutation.isPending}
    >
      <form
        className="space-y-5"
        onSubmit={form.handleSubmit((body) => mutation.mutate(body))}
      >
        <label className="field">
          Nome
          <input className="form-control" {...form.register('name')} />
          <FieldError message={form.formState.errors.name?.message} />
        </label>
        <label className="field">
          Referência
          <input className="form-control" {...form.register('reference')} />
          <FieldError message={form.formState.errors.reference?.message} />
        </label>
        <label className="flex items-center gap-2 text-sm">
          <input type="checkbox" {...form.register('active')} />
          Empresa ativa
        </label>
        <FormFooter
          pending={mutation.isPending}
          error={mutation.error}
          onCancel={onClose}
        />
      </form>
    </Dialog>
  )
}
